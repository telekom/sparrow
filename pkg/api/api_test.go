// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestAPI_Run(t *testing.T) {
	tests := []struct {
		name string
		want struct {
			method string
			path   string
			status int
		}
		wantErr bool
	}{
		{
			name: "Root route registered",
			want: struct {
				method string
				path   string
				status int
			}{
				method: http.MethodGet,
				path:   "/",
				status: http.StatusOK,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			a := api{
				server: &http.Server{Addr: ":8080"}, //nolint:gosec // irrelevant
				router: chi.NewRouter(),
			}

			if err := a.RegisterRoutes(ctx); err != nil {
				t.Fatalf("Failed to register routes: %v", err)
			}

			go func() {
				if err := a.Run(ctx); (err != nil) != tt.wantErr {
					t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()
			time.Sleep(10 * time.Millisecond)
			if !tt.wantErr {
				req := httptest.NewRequest(tt.want.method, tt.want.path, http.NoBody)
				rec := httptest.NewRecorder()
				a.router.ServeHTTP(rec, req)

				if status := rec.Result().StatusCode; status != tt.want.status {
					t.Errorf("Handler for route %s returned wrong status code: got %v want %v", tt.want.path, status, tt.want.status)
				}

				defer func() {
					err := rec.Result().Body.Close()
					if err != nil {
						t.Fatalf("Failed to close recoder body")
					}
				}()
				if err := a.Shutdown(ctx); err != nil {
					t.Fatalf("Failed to shutdown api: %v", err)
				}
			}
		})
	}
}

func TestAPI_RegisterRoutes(t *testing.T) {
	tests := []struct {
		name   string
		routes []Route
		want   []struct {
			method string
			path   string
			status int
		}
		wantErr bool
	}{
		{
			name: "Register one route",
			routes: []Route{
				{Path: "/get", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
			},
			want: []struct {
				method string
				path   string
				status int
			}{
				{method: http.MethodGet, path: "/get", status: http.StatusOK},
			},
			wantErr: false,
		},
		{
			name: "Register multiple routes",
			routes: []Route{
				{Path: "/get", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
				{Path: "/post", Method: http.MethodPost, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
				}},
				{Path: "/put", Method: http.MethodPut, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
				{Path: "/delete", Method: http.MethodDelete, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}},
				{Path: "/patch", Method: http.MethodPatch, Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
				{Path: "/handlefunc", Method: "*", Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}},
			},
			want: []struct {
				method string
				path   string
				status int
			}{
				{method: http.MethodGet, path: "/get", status: http.StatusOK},
				{method: http.MethodPost, path: "/post", status: http.StatusCreated},
				{method: http.MethodPut, path: "/put", status: http.StatusOK},
				{method: http.MethodDelete, path: "/delete", status: http.StatusNoContent},
				{method: http.MethodPatch, path: "/patch", status: http.StatusOK},
				{method: http.MethodGet, path: "/handlefunc", status: http.StatusOK},
			},
			wantErr: false,
		},
		{
			name: "Unsupported Method",
			routes: []Route{
				{Path: "/unknown", Method: "unknown", Handler: func(w http.ResponseWriter, r *http.Request) {}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := api{
				server: &http.Server{}, //nolint:gosec
				router: chi.NewRouter(),
			}

			err := a.RegisterRoutes(context.Background(), tt.routes...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterRoutes() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				for _, req := range tt.want {
					request := httptest.NewRequest(req.method, req.path, http.NoBody)
					recorder := httptest.NewRecorder()

					a.router.ServeHTTP(recorder, request)

					if recorder.Code != req.status {
						t.Errorf("Unexpected status code for %s %s. Got %d, wanted %d", req.method, req.path, recorder.Code, req.status)
					}
				}
			}
		})
	}
}

func TestAPI_ShutdownWhenContextCanceled(t *testing.T) {
	a := api{
		router: chi.NewRouter(),
		server: &http.Server{}, //nolint:gosec
	}
	ctx, cancel := context.WithCancel(context.Background())
	err := a.RegisterRoutes(ctx)
	if err != nil {
		t.Fatalf("Failed to register routes")
	}
	cancel()

	if err := a.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Error("Expected ErrApiContext")
	}
}

func TestAPI_OkHandler(t *testing.T) {
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, "GET", "/okHandler", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := OkHandler(ctx)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "ok"
	if rr.Body.String() != expected {
		t.Errorf("Handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{"Empty address", Config{}, true},
		{"Empty certpath", Config{Tls: TLSConfig{Enabled: true}}, true},
		{"Empty keypath", Config{Tls: TLSConfig{Enabled: true}}, true},

		{"Valid config", Config{ListeningAddress: ":8080"}, false},
		{"Valid tls config", Config{ListeningAddress: ":8080", Tls: TLSConfig{Enabled: true, CertPath: "./mycert.pem", KeyPath: "mykey.key"}}, false},
		{"Valid tls config without tls", Config{ListeningAddress: ":8080", Tls: TLSConfig{Enabled: false}}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.config.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
