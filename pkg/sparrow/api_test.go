package sparrow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
)

func TestSparrow_register(t *testing.T) {
	r := chi.NewRouter()
	s := Sparrow{
		router: r,
	}

	s.register()

	expectedRoutes := []string{"/openapi.yaml", "/v1/metrics/{checkName}", "/checks/*"}
	routes := r.Routes()
	for _, route := range expectedRoutes {
		found := 0

		for _, foundRoute := range routes {
			if foundRoute.Pattern == route {
				found += 1
				break
			}
		}
	}

}

func TestSparrow_api_shutdownWhenContextCanceled(t *testing.T) {
	s := Sparrow{
		cfg:    &config.Config{Api: config.ApiConfig{Port: ":8080"}},
		router: chi.NewRouter(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.api(ctx); !errors.Is(err, context.Canceled) {
		t.Error("Expected ErrApiContext")
	}

}

func TestSparrow_Openapi(t *testing.T) {
	type fields struct {
		checks      map[string]checks.Check
		routingTree api.RoutingTree
		resultFanIn map[string]chan checks.Result
		cResult     chan checks.ResultDTO
		loader      config.Loader
		cfg         *config.Config
		cCfgChecks  chan map[string]any
		router      chi.Router
		db          db.DB
	}
	tests := []struct {
		name    string
		fields  fields
		want    openapi3.T
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				checks:      tt.fields.checks,
				routingTree: tt.fields.routingTree,
				resultFanIn: tt.fields.resultFanIn,
				cResult:     tt.fields.cResult,
				loader:      tt.fields.loader,
				cfg:         tt.fields.cfg,
				cCfgChecks:  tt.fields.cCfgChecks,
				router:      tt.fields.router,
				db:          tt.fields.db,
			}
			got, err := s.Openapi()
			if (err != nil) != tt.wantErr {
				t.Errorf("Sparrow.Openapi() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sparrow.Openapi() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testDb() *db.InMemory {

	db := db.NewInMemory()
	db.Save(checks.ResultDTO{Name: "alpha", Result: &checks.Result{Timestamp: time.Now(), Err: "", Data: 1}})
	db.Save(checks.ResultDTO{Name: "beta", Result: &checks.Result{Timestamp: time.Now(), Err: "", Data: 1}})

	return db
}

func chiRequest(r *http.Request, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("checkName", value)

	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}
func TestSparrow_getCheckMetrics(t *testing.T) {
	type fields struct {
		checks      map[string]checks.Check
		routingTree api.RoutingTree
		resultFanIn map[string]chan checks.Result
		cResult     chan checks.ResultDTO
		loader      config.Loader
		cfg         *config.Config
		cCfgChecks  chan map[string]any
		router      chi.Router
		db          db.DB
	}

	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     []byte
		wantCode int
	}{
		{name: "no data", fields: fields{db: db.NewInMemory()}, args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/alpha", bytes.NewBuffer([]byte{})), "alpha")}, wantCode: http.StatusNotFound, want: []byte(http.StatusText(http.StatusNotFound))},
		{name: "bad request data", fields: fields{db: db.NewInMemory()}, args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/", bytes.NewBuffer([]byte{})), "")}, wantCode: http.StatusBadRequest, want: []byte(http.StatusText(http.StatusBadRequest))},
		{name: "has data", fields: fields{db: testDb()}, args: args{w: httptest.NewRecorder(), r: chiRequest(httptest.NewRequest(http.MethodGet, "/v1/metrics/alpha", bytes.NewBuffer([]byte{})), "alpha")}, wantCode: http.StatusOK, want: []byte(`{"name":"alpha","result":{"timestamp":"0001-01-01T00:00:00Z","err":"","data":1}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				checks:      tt.fields.checks,
				routingTree: tt.fields.routingTree,
				resultFanIn: tt.fields.resultFanIn,
				cResult:     tt.fields.cResult,
				loader:      tt.fields.loader,
				cfg:         tt.fields.cfg,
				cCfgChecks:  tt.fields.cCfgChecks,
				router:      tt.fields.router,
				db:          tt.fields.db,
			}
			s.getCheckMetrics(tt.args.w, tt.args.r)
			resp := tt.args.w.Result()
			body, _ := io.ReadAll(resp.Body)

			if tt.wantCode == http.StatusOK {
				if tt.wantCode != resp.StatusCode {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", resp.StatusCode, tt.wantCode)
				}
				var got checks.ResultDTO
				var want checks.ResultDTO
				err := json.Unmarshal(body, &got)
				if err != nil {
					t.Error("Expected valid json")
				}
				err = json.Unmarshal(tt.want, &want)
				if err != nil {
					t.Error("Expected valid json")
				}

				if reflect.DeepEqual(got, want) {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", got, want)
				}
			} else {
				if tt.wantCode != resp.StatusCode {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", resp.StatusCode, tt.wantCode)
				}
				if !reflect.DeepEqual(body, tt.want) {
					t.Errorf("Sparrow.getCheckMetrics() = %v, want %v", body, tt.want)
				}
			}
		})
	}
}

func TestSparrow_handleChecks(t *testing.T) {
	type route struct {
		Method  string
		Path    string
		Handler http.HandlerFunc
	}
	type fields struct {
		checks      map[string]checks.Check
		routingTree api.RoutingTree
		resultFanIn map[string]chan checks.Result
		cResult     chan checks.ResultDTO
		loader      config.Loader
		cfg         *config.Config
		cCfgChecks  chan map[string]any
		router      chi.Router
		routes      []route
		db          db.DB
	}
	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     []byte
		wantCode int
	}{
		{name: "no check handlers", fields: fields{routingTree: api.NewRoutingTree()}, args: args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/v1/notfound", bytes.NewBuffer([]byte{}))}, wantCode: http.StatusNotFound, want: []byte(http.StatusText(http.StatusNotFound))},
		{name: "has check handlers", fields: fields{routingTree: api.NewRoutingTree(), routes: []route{{Method: http.MethodGet, Path: "/v1/test", Handler: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("test")) }}}}, args: args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/v1/test", bytes.NewBuffer([]byte{}))}, wantCode: http.StatusOK, want: []byte("test")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				checks:      tt.fields.checks,
				routingTree: tt.fields.routingTree,
				resultFanIn: tt.fields.resultFanIn,
				cResult:     tt.fields.cResult,
				loader:      tt.fields.loader,
				cfg:         tt.fields.cfg,
				cCfgChecks:  tt.fields.cCfgChecks,
				router:      tt.fields.router,
				db:          tt.fields.db,
			}

			for _, route := range tt.fields.routes {
				s.routingTree.Add(route.Method, route.Path, route.Handler)
			}
			s.handleChecks(tt.args.w, tt.args.r)
			resp := tt.args.w.Result()
			body, _ := io.ReadAll(resp.Body)

			if tt.wantCode != resp.StatusCode {
				t.Errorf("Sparrow.handleChecks() = %v, want %v", resp.StatusCode, tt.wantCode)
			}
			if !reflect.DeepEqual(body, tt.want) {
				t.Errorf("Sparrow.handleChecks() = %v, want %v", body, tt.want)
			}

		})
	}
}
