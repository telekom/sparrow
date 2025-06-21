// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package targets

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/test"

	remotemock "github.com/telekom/sparrow/pkg/sparrow/targets/remote/test"
)

const (
	testCheckInterval        = 100 * time.Millisecond
	testRegistrationInterval = 150 * time.Millisecond
	testUpdateInterval       = 150 * time.Millisecond
)

// Test_gitlabTargetManager_refreshTargets tests that the refreshTargets method
// will fetch the targets from the remote instance and update the local
// targets list. When an unhealthyTheshold is set, it will also unregister
// unhealthy targets
func Test_gitlabTargetManager_refreshTargets(t *testing.T) {
	now := time.Now()
	tooOld := now.Add(-time.Hour * 2)

	tests := []struct {
		name               string
		mockTargets        []checks.GlobalTarget
		wantHealthy        []checks.GlobalTarget
		wantSelfRegistered bool
		wantErr            error
	}{
		{
			name:        "success with 0 targets",
			mockTargets: []checks.GlobalTarget{},
			wantHealthy: []checks.GlobalTarget{},
		},
		{
			name:               "success with 1 healthy target",
			mockTargets:        []checks.GlobalTarget{test.SparrowAZ1(t, now)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowAZ1(t, now)},
			wantSelfRegistered: false,
		},
		{
			name:               "success with 1 unhealthy target",
			mockTargets:        []checks.GlobalTarget{test.SparrowAZ1(t, tooOld)},
			wantSelfRegistered: false,
		},
		{
			name:               "success with 1 healthy and 1 unhealthy targets",
			mockTargets:        []checks.GlobalTarget{test.SparrowAZ1(t, now), test.SparrowAZ2(t, tooOld)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowAZ1(t, now)},
			wantSelfRegistered: false,
		},
		{
			name:               "success with self target",
			mockTargets:        []checks.GlobalTarget{test.SparrowLocal(t, now)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowLocal(t, now)},
			wantSelfRegistered: true,
		},
		{
			name:               "success with self target and 1 unhealthy target",
			mockTargets:        []checks.GlobalTarget{test.SparrowLocal(t, now), test.SparrowAZ1(t, tooOld)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowLocal(t, now)},
			wantSelfRegistered: true,
		},
		{
			name:        "failure getting targets",
			mockTargets: nil,
			wantHealthy: nil,
			wantErr:     errors.New("failed to fetch files"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := remotemock.New(tt.mockTargets...)
			if tt.wantErr != nil {
				remote.SetFetchFilesErr(tt.wantErr)
			}
			gtm := &manager{
				targets:    nil,
				interactor: remote,
				name:       test.SparrowLocalName,
				cfg:        General{UnhealthyThreshold: time.Hour, Scheme: "https"},
				metrics:    newMetrics(),
			}

			if err := gtm.refreshTargets(t.Context()); (err != nil) != (tt.wantErr != nil) {
				t.Errorf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, gtm.registered, tt.wantSelfRegistered, "refreshTargets() did not find self target")

			require.Len(t, gtm.targets, len(tt.wantHealthy))
			for i := range gtm.targets {
				assert.Equal(t, gtm.targets[i].URL, tt.wantHealthy[i].URL)
				assert.Equal(t, gtm.targets[i].LastSeen, tt.wantHealthy[i].LastSeen)
			}
		})
	}
}

// Test_gitlabTargetManager_refreshTargets_No_Threshold tests that the
// refreshTargets method will not unregister unhealthy targets if the
// unhealthyThreshold is 0
func Test_gitlabTargetManager_refreshTargets_No_Threshold(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-time.Hour * 24)

	tests := []struct {
		name               string
		mockTargets        []checks.GlobalTarget
		wantHealthy        []checks.GlobalTarget
		wantSelfRegistered bool
		wantErr            error
	}{
		{
			name:               "success with 1 target",
			mockTargets:        []checks.GlobalTarget{test.SparrowAZ1(t, yesterday)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowAZ1(t, now.Add(-time.Hour*2))},
			wantSelfRegistered: false,
		},
		{
			name:               "success with 2 old targets",
			mockTargets:        []checks.GlobalTarget{test.SparrowAZ1(t, yesterday), test.SparrowAZ2(t, yesterday)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowAZ1(t, yesterday), test.SparrowAZ2(t, yesterday)},
			wantSelfRegistered: false,
		},
		{
			name:               "success with self target",
			mockTargets:        []checks.GlobalTarget{test.SparrowLocal(t, now)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowLocal(t, now)},
			wantSelfRegistered: true,
		},
		{
			name:               "success with self old target",
			mockTargets:        []checks.GlobalTarget{test.SparrowLocal(t, yesterday)},
			wantHealthy:        []checks.GlobalTarget{test.SparrowLocal(t, now.Add(-time.Hour*2))},
			wantSelfRegistered: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := remotemock.New(tt.mockTargets...)
			gtm := &manager{
				targets:    nil,
				interactor: remote,
				name:       test.SparrowLocalName,
				cfg:        General{UnhealthyThreshold: 0, Scheme: "https"},
				metrics:    newMetrics(),
			}

			if err := gtm.refreshTargets(t.Context()); (err != nil) != (tt.wantErr != nil) {
				t.Errorf("refreshTargets() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, gtm.registered, tt.wantSelfRegistered, "refreshTargets() did not find self target")

			require.Len(t, gtm.targets, len(tt.wantHealthy))
			for i := range gtm.targets {
				assert.Equal(t, gtm.targets[i].URL, tt.wantHealthy[i].URL)
			}
		})
	}
}

func Test_gitlabTargetManager_GetTargets(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		targets []checks.GlobalTarget
		want    []checks.GlobalTarget
	}{
		{
			name:    "success with 0 targets",
			targets: nil,
			want:    nil,
		},
		{
			name:    "success with 1 target",
			targets: []checks.GlobalTarget{test.SparrowAZ1(t, now)},
			want:    []checks.GlobalTarget{test.SparrowAZ1(t, now)},
		},
		{
			name:    "success with 2 targets",
			targets: []checks.GlobalTarget{test.SparrowAZ1(t, now), test.SparrowAZ2(t, now)},
			want:    []checks.GlobalTarget{test.SparrowAZ1(t, now), test.SparrowAZ2(t, now)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtm := &manager{
				targets: tt.targets,
			}
			got := gtm.GetTargets()

			require.Len(t, got, len(tt.want))
			for i := range got {
				assert.Equal(t, got[i].URL, tt.want[i].URL)
				assert.Equal(t, got[i].LastSeen, tt.want[i].LastSeen)
			}
		})
	}
}

// Test_gitlabTargetManager_registerSparrow tests that the register method will
// register the sparrow instance in the remote instance
func Test_gitlabTargetManager_register(t *testing.T) {
	tests := []struct {
		name       string
		wantErr    bool
		wantPutErr bool
	}{
		{
			name: "success",
		},
		{
			name:       "failure - failed to register",
			wantErr:    true,
			wantPutErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := remotemock.New()
			if tt.wantPutErr {
				glmock.SetPostFileErr(errors.New("failed to register"))
			}

			gtm := &manager{
				interactor: glmock,
				metrics:    newMetrics(),
				name:       test.SparrowLocalName,
				cfg:        General{Scheme: "https"},
			}
			if err := gtm.register(t.Context()); (err != nil) != tt.wantErr {
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				assert.True(t, gtm.registered, "register() did not register the instance")
			}
		})
	}
}

// Test_gitlabTargetManager_update tests that the update
// method will update the registration of the sparrow instance in the remote instance
func Test_gitlabTargetManager_update(t *testing.T) {
	tests := []struct {
		name         string
		wantPutError bool
	}{
		{
			name: "success - update registration",
		},
		{
			name:         "failure - failed to update registration",
			wantPutError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := remotemock.New()
			if tt.wantPutError {
				glmock.SetPutFileErr(fmt.Errorf("failed to update registration"))
			}

			gtm := &manager{
				interactor: glmock,
				registered: true,
				cfg:        General{Scheme: "https"},
			}

			wantErr := tt.wantPutError
			if err := gtm.update(t.Context()); (err != nil) != wantErr {
				t.Errorf("update() error = %v, wantErr %v", err, wantErr)
			}
		})
	}
}

// Test_gitlabTargetManager_Reconcile_success tests that the Reconcile method
// will register the target if it is not registered yet and update the
// registration if it is already registered
func Test_gitlabTargetManager_Reconcile_success(t *testing.T) {
	tests := []struct {
		name          string
		registered    bool
		wantPostError bool
		wantPutError  bool
	}{
		{
			name: "success - first registration",
		},
		{
			name:       "success - update registration",
			registered: true,
		},
	}

	testTarget := test.SparrowAZ1(t, time.Now())
	glmock := remotemock.New(testTarget)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gtm := mockGitlabTargetManager(glmock)
			gtm.registered = tt.registered
			ctx := t.Context()
			go func() { require.NoError(t, gtm.Reconcile(ctx)) }()
			defer func() { require.NoError(t, gtm.Shutdown(ctx)) }()

			<-time.After(testUpdateInterval * 2)
			require.Len(t, gtm.GetTargets(), 1)
			assert.Equal(t, gtm.GetTargets()[0].URL, testTarget.URL)

			gtm.mu.Lock()
			assert.True(t, gtm.registered, "Reconcile() should have registered the sparrow")
			gtm.mu.Unlock()
		})
	}
}

// Test_gitlabTargetManager_Reconcile_Registration_Update tests that the Reconcile
// method will register the sparrow, and then update the registration after the
// registration interval has passed
func Test_gitlabTargetManager_Reconcile_Registration_Update(t *testing.T) {
	glmock := remotemock.New(test.SparrowAZ1(t, time.Now()))

	gtm := mockGitlabTargetManager(glmock)
	gtm.cfg.RegistrationInterval = 10 * time.Millisecond
	gtm.cfg.UpdateInterval = 100 * time.Millisecond

	ctx := t.Context()
	go func() { require.NoError(t, gtm.Reconcile(ctx)) }()
	defer func() { require.NoError(t, gtm.Shutdown(ctx)) }()

	<-time.After(gtm.cfg.UpdateInterval * 3)
	gtm.mu.Lock()
	assert.True(t, gtm.registered, "Reconcile() should be registered")
	gtm.mu.Unlock()

	// check that the post call was made once, to create the registration
	assert.Equal(t, glmock.PostFileCount(), 1, "Reconcile() should have registered the instance once")

	// check that the put call was made twice, to update the registration
	assert.Equal(t, glmock.PutFileCount(), 2, "Reconcile() should have updated the registration twice")

	t.Logf("Reconcile() successfully registered and updated the sparrow")
}

// Test_gitlabTargetManager_Reconcile_failure tests that the Reconcile method
// will handle API failures gracefully
func Test_gitlabTargetManager_Reconcile_failure(t *testing.T) {
	tests := []struct {
		name       string
		registered bool
		targets    []checks.GlobalTarget
		postErr    error
		putError   error
	}{
		{
			name:    "failure - failed to register",
			postErr: errors.New("failed to register"),
		},
		{
			name:       "failure - failed to update registration",
			registered: true,
			putError:   errors.New("failed to update registration"),
			targets: []checks.GlobalTarget{
				test.SparrowAZ1(t, time.Now()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glmock := remotemock.New(tt.targets...)

			gtm := mockGitlabTargetManager(glmock)
			glmock.SetPostFileErr(tt.postErr)
			glmock.SetPutFileErr(tt.putError)

			ctx := t.Context()
			go func() { require.NoError(t, gtm.Reconcile(ctx)) }()
			defer func() { require.NoError(t, gtm.Shutdown(ctx)) }()

			<-time.After(300 * time.Millisecond)

			gtm.mu.Lock()
			if tt.postErr != nil {
				assert.False(t, gtm.registered, "Reconcile() should not have registered")
			}

			if tt.putError != nil {
				assert.True(t, gtm.registered, "Reconcile() should still be registered")
			}
			gtm.mu.Unlock()
		})
	}
}

// Test_gitlabTargetManager_Reconcile_Context_Canceled tests that the Reconcile
// method will shutdown gracefully when the context is canceled.
func Test_gitlabTargetManager_Reconcile_Context_Canceled(t *testing.T) {
	glmock := remotemock.New(
		test.SparrowAZ1(t, time.Now()),
	)

	gtm := mockGitlabTargetManager(glmock)

	ctx, cancel := context.WithCancel(t.Context())
	go func() { require.ErrorIs(t, gtm.Reconcile(ctx), context.Canceled) }()

	<-time.After(250 * time.Millisecond)
	cancel()
	<-time.After(250 * time.Millisecond)

	gtm.mu.Lock()
	assert.True(t, gtm.registered, "Reconcile() should still be registered")
	gtm.mu.Unlock()

	if !glmock.PostFileCalled() || !glmock.PutFileCalled() {
		t.Error("Reconcile() should have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_Context_Done tests that the Reconcile
// method will shut down gracefully when the context is done.
func Test_gitlabTargetManager_Reconcile_Context_Done(t *testing.T) {
	glmock := remotemock.New(test.SparrowAZ1(t, time.Now()))

	gtm := mockGitlabTargetManager(glmock)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	go func() { require.ErrorIs(t, gtm.Reconcile(ctx), context.DeadlineExceeded) }()

	<-time.After(15 * time.Millisecond)

	gtm.mu.Lock()
	assert.False(t, gtm.registered, "Reconcile() should not be registered")
	gtm.mu.Unlock()

	if glmock.PostFileCalled() || glmock.PutFileCalled() {
		t.Error("Reconcile() should not have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_Shutdown tests that the Reconcile
// method will shut down gracefully when the Shutdown method is called.
func Test_gitlabTargetManager_Reconcile_Shutdown(t *testing.T) {
	glmock := remotemock.New(test.SparrowAZ1(t, time.Now()))

	gtm := mockGitlabTargetManager(glmock)

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	go func() { require.NoError(t, gtm.Reconcile(ctx)) }()

	<-time.After(250 * time.Millisecond)

	require.NoError(t, gtm.Shutdown(ctx), "Reconcile() failed to shutdown gracefully")

	gtm.mu.Lock()
	assert.False(t, gtm.registered, "Reconcile() should not be registered")
	gtm.mu.Unlock()

	if !glmock.PostFileCalled() || !glmock.PutFileCalled() {
		t.Error("Reconcile() should have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_Shutdown_Fail_Unregister tests that the Reconcile
// method will fail the graceful shutdown when the Shutdown method is called
// and the unregistering fails.
func Test_gitlabTargetManager_Reconcile_Shutdown_Fail_Unregister(t *testing.T) {
	glmock := remotemock.New(test.SparrowAZ1(t, time.Now()))

	gtm := mockGitlabTargetManager(glmock)
	glmock.SetDeleteFileErr(errors.New("gitlab API error"))

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	go func() { require.ErrorIs(t, gtm.Reconcile(ctx), context.Canceled) }()

	<-time.After(250 * time.Millisecond)

	require.Error(t, gtm.Shutdown(ctx), "Reconcile() should have failed to shutdown")

	gtm.mu.Lock()
	assert.True(t, gtm.registered, "Reconcile() should still be registered")
	gtm.mu.Unlock()

	if !glmock.PostFileCalled() || !glmock.PutFileCalled() {
		t.Error("Reconcile() should have made calls to the gitlab API")
	}
}

// Test_gitlabTargetManager_Reconcile_No_Registration tests that the Reconcile
// method will not register the instance if the registration interval is 0
func Test_gitlabTargetManager_Reconcile_No_Registration(t *testing.T) {
	glmock := remotemock.New(test.SparrowAZ1(t, time.Now()))

	gtm := mockGitlabTargetManager(glmock)
	gtm.cfg.RegistrationInterval = 0

	ctx := t.Context()
	go func() { require.NoError(t, gtm.Reconcile(ctx)) }()
	defer func() { require.NoError(t, gtm.Shutdown(ctx)) }()

	<-time.After(250 * time.Millisecond)

	gtm.mu.Lock()
	assert.False(t, gtm.registered, "Reconcile() should not be registered")
	gtm.mu.Unlock()

	if glmock.PostFileCalled() || glmock.PutFileCalled() {
		t.Error("Reconcile() should not have registered the instance")
	}
}

// Test_gitlabTargetManager_Reconcile_No_Update tests that the Reconcile
// method will not update the registration if the update interval is 0
func Test_gitlabTargetManager_Reconcile_No_Update(t *testing.T) {
	glmock := remotemock.New(test.SparrowAZ1(t, time.Now()))

	gtm := mockGitlabTargetManager(glmock)
	gtm.cfg.UpdateInterval = 0

	ctx := t.Context()
	go func() { require.NoError(t, gtm.Reconcile(ctx)) }()
	defer func() { require.NoError(t, gtm.Shutdown(ctx)) }()

	<-time.After(250 * time.Millisecond)

	gtm.mu.Lock()
	assert.True(t, gtm.registered, "Reconcile() should be registered")
	gtm.mu.Unlock()

	assert.False(t, glmock.PutFileCalled(), "Reconcile() should not have registered the instance")
	assert.Equal(t, glmock.PostFileCount(), 1, "Reconcile() should have registered the instance")
}

// Test_gitlabTargetManager_Reconcile_No_Registration_No_Update tests that the Reconcile
// method will not register the instance if the registration interval is 0
// and will not update the registration if the update interval is 0
func Test_gitlabTargetManager_Reconcile_No_Registration_No_Update(t *testing.T) {
	glmock := remotemock.New(test.SparrowAZ1(t, time.Now()))

	gtm := mockGitlabTargetManager(glmock)
	gtm.cfg.RegistrationInterval = 0
	gtm.cfg.UpdateInterval = 0

	ctx := t.Context()
	go func() { require.NoError(t, gtm.Reconcile(ctx)) }()
	defer func() { require.NoError(t, gtm.Shutdown(ctx)) }()

	<-time.After(250 * time.Millisecond)

	gtm.mu.Lock()
	assert.False(t, gtm.registered, "Reconcile() should not be registered")
	gtm.mu.Unlock()

	assert.False(t, glmock.PostFileCalled(), "Reconcile() should not have registered the instance")
	assert.False(t, glmock.PutFileCalled(), "Reconcile() should not have updated the registration")
}

func mockGitlabTargetManager(g *remotemock.MockClient) *manager {
	return &manager{
		targets:    nil,
		done:       make(chan struct{}, 1),
		interactor: g,
		name:       test.SparrowLocalName,
		metrics:    newMetrics(),
		cfg: General{
			CheckInterval:        100 * time.Millisecond,
			UnhealthyThreshold:   1 * time.Second,
			RegistrationInterval: testRegistrationInterval,
			UpdateInterval:       testUpdateInterval,
			Scheme:               "https",
		},
		registered: false,
	}
}
