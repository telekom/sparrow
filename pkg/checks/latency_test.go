package checks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/caas-team/sparrow/pkg/api"
	"github.com/jarcoal/httpmock"
)

func stringPointer(s string) *string {
	return &s
}

func TestLatency_Run(t *testing.T) {
	ctx := context.Background()
	httpmock.Activate()
	// defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(http.MethodGet, "http://success.com", httpmock.NewStringResponder(200, "ok"))
	httpmock.RegisterResponder(http.MethodGet, "http://fail.com", httpmock.NewStringResponder(500, "fail"))
	httpmock.RegisterResponder(http.MethodGet, "http://timeout.com", httpmock.NewErrorResponder(context.DeadlineExceeded))

	c := NewLatencyCheck()
	results := make(chan Result, 1)
	c.Startup(ctx, results)

	c.SetConfig(ctx, LatencyConfig{
		Targets:  []string{"http://success.com", "http://fail.com", "http://timeout.com"},
		Interval: time.Second * 120,
		Timeout:  time.Second * 5,
	})

	go c.Run(ctx)
	defer c.Shutdown(ctx)

	result := <-results

	wantResult := Result{
		Timestamp: result.Timestamp,
		Err:       "",
		Data: map[string]LatencyResult{
			"http://success.com": {
				Code:  200,
				Error: nil,
				Total: 0,
			},
			"http://fail.com": {
				Code:  500,
				Error: nil,
				Total: 0,
			},
			"http://timeout.com": {
				Code:  0,
				Error: stringPointer("Get \"http://timeout.com\": context deadline exceeded"),
				Total: 0,
			},
		},
	}

	if wantResult.Timestamp != result.Timestamp {
		t.Errorf("Latency.Run() = %v, want %v", result.Timestamp, wantResult.Timestamp)
	}
	if wantResult.Err != result.Err {
		t.Errorf("Latency.Run() = %v, want %v", result.Err, wantResult.Err)
	}
	wantData := wantResult.Data.(map[string]LatencyResult)
	data := result.Data.(map[string]LatencyResult)

	for k, v := range wantData {
		if v.Code != data[k].Code {
			t.Errorf("Latency.Run() = %v, want %v", data[k].Code, v.Code)
		}
		if data[k].Total < 0 {
			t.Errorf("Latency.Run() = %v, want < 0", data[k].Total)
		}
		if v.Error != nil && data[k].Error != nil {
			if *v.Error != *data[k].Error {
				t.Errorf("Latency.Run() = %v, want %v", *data[k].Error, *v.Error)
			}
		}
	}
}

func TestLatency_check(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.Deactivate)

	tests := []struct {
		name                string
		registeredEndpoints []struct {
			name    string
			status  int
			success bool
		}
		targets []string
		ctx     context.Context
		want    map[string]LatencyResult
	}{
		{
			name:                "no target",
			registeredEndpoints: nil,
			targets:             []string{},
			ctx:                 context.Background(),
			want:                map[string]LatencyResult{},
		},
		{
			name: "one target",
			registeredEndpoints: []struct {
				name    string
				status  int
				success bool
			}{
				{
					name:    "http://success.com",
					status:  200,
					success: true,
				},
			},
			targets: []string{"http://success.com"},
			ctx:     context.Background(),
			want: map[string]LatencyResult{
				"http://success.com": {Code: http.StatusOK, Error: nil, Total: 0},
			},
		},
		{
			name: "multiple targets",
			registeredEndpoints: []struct {
				name    string
				status  int
				success bool
			}{
				{
					name:    "http://success.com",
					status:  http.StatusOK,
					success: true,
				},
				{
					name:    "http://fail.com",
					status:  http.StatusInternalServerError,
					success: true,
				},
				{
					name:    "http://timeout.com",
					success: false,
				},
			},
			targets: []string{"http://success.com", "http://fail.com", "http://timeout.com"},
			ctx:     context.Background(),
			want: map[string]LatencyResult{
				"http://success.com": {
					Code:  200,
					Error: nil,
					Total: 0,
				},
				"http://fail.com": {
					Code:  500,
					Error: nil,
					Total: 0,
				},
				"http://timeout.com": {
					Code:  0,
					Error: stringPointer("Get \"http://timeout.com\": context deadline exceeded"),
					Total: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, endpoint := range tt.registeredEndpoints {
				if endpoint.success {
					httpmock.RegisterResponder(http.MethodGet, endpoint.name, httpmock.NewStringResponder(endpoint.status, ""))
				} else {
					httpmock.RegisterResponder(http.MethodGet, endpoint.name, httpmock.NewErrorResponder(context.DeadlineExceeded))
				}
			}

			l := &Latency{
				cfg: LatencyConfig{Targets: tt.targets, Interval: time.Second * 120, Timeout: time.Second * 1},
			}

			got, err := l.check(tt.ctx)
			if err != nil {
				t.Errorf("check() error = %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("check() got %v results, want %v results", len(got), len(tt.want))
			}

			for k, v := range tt.want {
				if v.Code != got[k].Code {
					t.Errorf("Latency.check() = %v, want %v", got[k].Code, v.Code)
				}
				if got[k].Total < 0 {
					t.Errorf("Latency.check() got negative latency for key %v", k)
				}
				if v.Error != nil && got[k].Error != nil {
					if *v.Error != *got[k].Error {
						t.Errorf("Latency.check() = %v, want %v", *got[k].Error, *v.Error)
					}
				}
			}

			// Resetting httpmock for the next iteration
			httpmock.Reset()
		})
	}

}

func TestLatency_Startup(t *testing.T) {
	c := Latency{}

	if err := c.Startup(context.Background(), make(chan<- Result, 1)); err != nil {
		t.Errorf("Startup() error = %v", err)
	}
}

func TestLatency_Shutdown(t *testing.T) {
	cDone := make(chan bool, 1)
	c := Latency{
		done: cDone,
	}
	err := c.Shutdown(context.Background())

	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if !<-cDone {
		t.Error("Shutdown() should be ok")
	}

}

func TestLatency_SetConfig(t *testing.T) {
	c := Latency{}
	wantCfg := LatencyConfig{
		Targets: []string{"http://localhost:9090"},
	}

	err := c.SetConfig(context.Background(), wantCfg)

	if err != nil {
		t.Errorf("SetConfig() error = %v", err)
	}
	if !reflect.DeepEqual(c.cfg, wantCfg) {
		t.Errorf("SetConfig() = %v, want %v", c.cfg, wantCfg)
	}
}

func TestLatency_RegisterHandler(t *testing.T) {
	c := Latency{}

	rt := api.NewRoutingTree()
	c.RegisterHandler(context.Background(), &rt)

	h, ok := rt.Get("GET", "v1alpha1/latency")

	if !ok {
		t.Error("RegisterHandler() should be ok")
	}
	if h == nil {
		t.Error("RegisterHandler() should not be nil")
	}
	c.DeregisterHandler(context.Background(), &rt)
	h, ok = rt.Get("GET", "v1alpha1/latency")

	if ok {
		t.Error("DeregisterHandler() should not be ok")
	}

	if h != nil {
		t.Error("DeregisterHandler() should be nil")
	}

}

func TestLatency_Handler(t *testing.T) {
	c := Latency{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1alpha1/latency", nil)

	c.Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Handler() should be ok, got %d", rec.Code)
	}

}

func TestNewLatencyCheck(t *testing.T) {
	c := NewLatencyCheck()
	if c == nil {
		t.Error("NewLatencyCheck() should not be nil")
	}
}
