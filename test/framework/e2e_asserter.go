// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telekom/sparrow/pkg/checks"
)

// e2eHttpAsserter is an HTTP asserter for end-to-end tests.
type e2eHttpAsserter struct {
	e2e      *E2E
	url      string
	response *e2eResponseAsserter
	schema   *openapi3.T
	router   routers.Router
}

// e2eResponseAsserter holds the expected response result and an asserter function.
type e2eResponseAsserter struct {
	want     any
	asserter func(r *http.Response) error
}

// HttpAssertion creates a new HTTP assertion for the given URL.
func (e *E2E) HttpAssertion(u string) *e2eHttpAsserter {
	return &e2eHttpAsserter{e2e: e, url: u}
}

// Assert asserts the status code and then runs schema and check result validations.
func (a *e2eHttpAsserter) Assert(status int) {
	a.e2e.t.Helper()
	if !a.e2e.isRunning() {
		a.e2e.t.Fatal("e2eHttpAsserter.Assert must be called after E2E.Run")
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, a.url, http.NoBody)
	if err != nil {
		a.e2e.t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.e2e.t.Errorf("Failed to get %s: %v", a.url, err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(a.e2e.t, status, resp.StatusCode, "Unexpected status code for %s", a.url)
	a.e2e.t.Logf("Got status code %d for %s", resp.StatusCode, a.url)

	if resp.StatusCode == http.StatusOK {
		if a.schema != nil && a.router != nil {
			if err = a.assertSchema(req, resp); err != nil {
				a.e2e.t.Errorf("Response from %q does not match schema: %v", a.url, err)
			}
		}

		if a.response != nil {
			if err = a.response.asserter(resp); err != nil {
				a.e2e.t.Errorf("Failed to assert response: %v", err)
			}
		}
	}
}

// WithSchema fetches the OpenAPI schema and creates a router for response validation.
func (a *e2eHttpAsserter) WithSchema() *e2eHttpAsserter {
	a.e2e.t.Helper()
	schema, err := a.fetchSchema()
	if err != nil {
		a.e2e.t.Fatalf("Failed to fetch OpenAPI schema: %v", err)
	}

	router, err := gorillamux.NewRouter(schema)
	if err != nil {
		a.e2e.t.Fatalf("Failed to create router from OpenAPI schema: %v", err)
	}

	a.schema = schema
	a.router = router
	return a
}

// WithCheckResult sets the expected check result and uses a custom asserter.
func (a *e2eHttpAsserter) WithCheckResult(r checks.Result) *e2eHttpAsserter {
	a.e2e.t.Helper()
	a.response = &e2eResponseAsserter{
		want:     r,
		asserter: a.assertCheckResponse,
	}
	return a
}

// fetchSchema retrieves the OpenAPI schema from the server.
func (a *e2eHttpAsserter) fetchSchema() (*openapi3.T, error) {
	ctx := context.Background()
	u, err := url.Parse(a.url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	u.Path = "/openapi"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to GET OpenAPI schema: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI schema: %w", err)
	}

	loader := openapi3.NewLoader()
	schema, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI schema: %w", err)
	}

	if err = schema.Validate(ctx); err != nil {
		return nil, fmt.Errorf("OpenAPI schema validation error: %w", err)
	}

	return schema, nil
}

// assertSchema validates the response body against the OpenAPI schema.
func (a *e2eHttpAsserter) assertSchema(req *http.Request, resp *http.Response) error {
	route, _, err := a.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("failed to find route: %w", err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	// Reset resp.Body so that further reading is possible.
	resp.Body = io.NopCloser(bytes.NewReader(data))

	responseRef := route.Operation.Responses.Status(resp.StatusCode)
	if responseRef == nil || responseRef.Value == nil {
		return fmt.Errorf("no response defined in OpenAPI schema for status code %d", resp.StatusCode)
	}

	mediaType := responseRef.Value.Content.Get("application/json")
	if mediaType == nil {
		return errors.New("no media type defined in OpenAPI schema for Content-Type 'application/json'")
	}

	var body map[string]any
	if err = json.Unmarshal(data, &body); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Validate the response body against the schema.
	if err = mediaType.Schema.Value.VisitJSON(body); err != nil {
		return fmt.Errorf("response body does not match schema: %w", err)
	}

	return nil
}

// assertCheckResponse decodes the check result from the response and compares it against the expected result.
func (a *e2eHttpAsserter) assertCheckResponse(resp *http.Response) error {
	want, ok := a.response.want.(checks.Result)
	require.True(a.e2e.t, ok, "Invalid response type: %T", a.response.want)

	var got checks.Result
	err := json.NewDecoder(resp.Body).Decode(&got)
	require.NoError(a.e2e.t, err, "Failed to decode response body")

	// Use a helper to compare the check results.
	assertCheckResult(a.e2e.t, want, got)
	return nil
}

// e2eTimeMargin defines the acceptable time margin for end-to-end tests.
const e2eTimeMargin = 5 * time.Minute

// assertCheckResult compares expected and actual check results.
// It verifies the data maps using type-specific assertions and checks that the timestamp is recent.
func assertCheckResult(t *testing.T, expected, actual checks.Result) {
	expMap, ok := expected.Data.(map[string]any)
	require.True(t, ok, "Expected Data is not a map, got %T", expected.Data)
	actMap, ok := actual.Data.(map[string]any)
	require.True(t, ok, "Actual Data is not a map, got %T", actual.Data)

	assertMapEqual(t, expMap, actMap)
	assert.Less(t, time.Since(actual.Timestamp), e2eTimeMargin, "Response timestamp is not recent: %v", actual.Timestamp)
}

// assertMapEqual iterates over the expected map keys and compares values using assertValueEqual.
func assertMapEqual(t *testing.T, expected, actual map[string]any) {
	for key, expVal := range expected {
		actVal, exists := actual[key]
		assert.True(t, exists, "Missing key %s in actual data", key)
		assertValueEqual(t, expVal, actVal)
	}
	assert.Len(t, actual, len(expected), "Map lengths differ")
}

// assertValueEqual performs type-specific comparisons for common edge cases.
// We cannot simply use [assert.Equal] or [reflect.DeepEqual] because certain types
// (such as timestamps, floats, or IP addresses) require a more nuanced comparison.
// This helper function ensures that we allow for acceptable margins and conversions.
func assertValueEqual(t *testing.T, expected, actual any) {
	switch exp := expected.(type) {
	// For maps, we need to recursively compare each key-value pair.
	case map[string]any:
		act, ok := actual.(map[string]any)
		require.True(t, ok, "Expected value for map is not a map, got %T", actual)
		assertMapEqual(t, exp, act)

	// For time.Time values, we cannot expect an exact match due to slight timing differences.
	case time.Time:
		act, ok := actual.(time.Time)
		require.True(t, ok, "Expected time.Time, got %T", actual)
		// Instead of a strict equality, we ensure the actual timestamp is within an acceptable margin.
		assert.WithinDuration(t, time.Now(), act, e2eTimeMargin, "Timestamp is not recent")

	// For floating point numbers, allow a small delta because floating-point arithmetic
	// can introduce minor variations which are acceptable in our time-sensitive context.
	case float32, float64:
		expFloat := toFloat64(exp)
		actFloat := toFloat64(actual)
		margin := time.Since(time.Now().Add(-e2eTimeMargin)).Seconds()
		assert.InDelta(t, expFloat, actFloat, margin, "Time-sensitive float values differ by more than %v", margin)

	// JSON unmarshals numbers as float64, so integers must be converted before comparison.
	case int:
		expFloat := toFloat64(exp)
		actFloat := toFloat64(actual)
		assert.Equal(t, expFloat, actFloat, "Int value differs")

	// For slices of strings, we need to account for potential IP address comparisons.
	case []string:
		// We need to convert the actual slice to a slice of strings before comparing elements.
		// Unfortunately, we cannot use a type assertion directly because the slice type is []any.
		actSlice, ok := actual.([]any)
		require.True(t, ok, "Expected slice of any for []string, got %T", actual)

		var actStrings []string
		for i, v := range actSlice {
			s, ok := v.(string)
			require.True(t, ok, "Element at index %d is not a string, got %T", i, v)
			actStrings = append(actStrings, s)
		}

		// Compare each element. If the expected string is an IP address, verify the actual string is a valid IP.
		for i, expStr := range exp {
			if ip := net.ParseIP(expStr); ip != nil {
				require.Less(t, i, len(actStrings), "Index out of range for IP slice")
				actIP := net.ParseIP(actStrings[i])
				require.NotNil(t, actIP, "Actual value at index %d is not a valid IP", i)
			} else {
				// For non-IP strings, a simple equality check is sufficient.
				assert.Equal(t, expStr, actStrings[i], "String at index %d differs", i)
			}
		}

	// For all other types, fall back to a simple equality check.
	default:
		assert.Equal(t, expected, actual, "Values differ")
	}
}

// toFloat64 converts various numeric types to float64.
// This helper is necessary because JSON unmarshals numeric values as float64,
// and we need a common type for comparison.
func toFloat64(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		// If the type is not a recognized numeric type, return 0.
		return 0
	}
}
