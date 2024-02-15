// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package api

import (
	"context"
	"sync"
)

// Ensure, that APIMock does implement API.
// If this is not the case, regenerate this file with moq.
var _ API = &APIMock{}

// APIMock is a mock implementation of API.
//
//	func TestSomethingThatUsesAPI(t *testing.T) {
//
//		// make and configure a mocked API
//		mockedAPI := &APIMock{
//			RegisterRoutesFunc: func(ctx context.Context, routes ...Route) error {
//				panic("mock out the RegisterRoutes method")
//			},
//			RunFunc: func(ctx context.Context) error {
//				panic("mock out the Run method")
//			},
//			ShutdownFunc: func(ctx context.Context) error {
//				panic("mock out the Shutdown method")
//			},
//		}
//
//		// use mockedAPI in code that requires API
//		// and then make assertions.
//
//	}
type APIMock struct {
	// RegisterRoutesFunc mocks the RegisterRoutes method.
	RegisterRoutesFunc func(ctx context.Context, routes ...Route) error

	// RunFunc mocks the Run method.
	RunFunc func(ctx context.Context) error

	// ShutdownFunc mocks the Shutdown method.
	ShutdownFunc func(ctx context.Context) error

	// calls tracks calls to the methods.
	calls struct {
		// RegisterRoutes holds details about calls to the RegisterRoutes method.
		RegisterRoutes []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Routes is the routes argument value.
			Routes []Route
		}
		// Run holds details about calls to the Run method.
		Run []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// Shutdown holds details about calls to the Shutdown method.
		Shutdown []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
	}
	lockRegisterRoutes sync.RWMutex
	lockRun            sync.RWMutex
	lockShutdown       sync.RWMutex
}

// RegisterRoutes calls RegisterRoutesFunc.
func (mock *APIMock) RegisterRoutes(ctx context.Context, routes ...Route) error {
	if mock.RegisterRoutesFunc == nil {
		panic("APIMock.RegisterRoutesFunc: method is nil but API.RegisterRoutes was just called")
	}
	callInfo := struct {
		Ctx    context.Context
		Routes []Route
	}{
		Ctx:    ctx,
		Routes: routes,
	}
	mock.lockRegisterRoutes.Lock()
	mock.calls.RegisterRoutes = append(mock.calls.RegisterRoutes, callInfo)
	mock.lockRegisterRoutes.Unlock()
	return mock.RegisterRoutesFunc(ctx, routes...)
}

// RegisterRoutesCalls gets all the calls that were made to RegisterRoutes.
// Check the length with:
//
//	len(mockedAPI.RegisterRoutesCalls())
func (mock *APIMock) RegisterRoutesCalls() []struct {
	Ctx    context.Context
	Routes []Route
} {
	var calls []struct {
		Ctx    context.Context
		Routes []Route
	}
	mock.lockRegisterRoutes.RLock()
	calls = mock.calls.RegisterRoutes
	mock.lockRegisterRoutes.RUnlock()
	return calls
}

// Run calls RunFunc.
func (mock *APIMock) Run(ctx context.Context) error {
	if mock.RunFunc == nil {
		panic("APIMock.RunFunc: method is nil but API.Run was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockRun.Lock()
	mock.calls.Run = append(mock.calls.Run, callInfo)
	mock.lockRun.Unlock()
	return mock.RunFunc(ctx)
}

// RunCalls gets all the calls that were made to Run.
// Check the length with:
//
//	len(mockedAPI.RunCalls())
func (mock *APIMock) RunCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockRun.RLock()
	calls = mock.calls.Run
	mock.lockRun.RUnlock()
	return calls
}

// Shutdown calls ShutdownFunc.
func (mock *APIMock) Shutdown(ctx context.Context) error {
	if mock.ShutdownFunc == nil {
		panic("APIMock.ShutdownFunc: method is nil but API.Shutdown was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockShutdown.Lock()
	mock.calls.Shutdown = append(mock.calls.Shutdown, callInfo)
	mock.lockShutdown.Unlock()
	return mock.ShutdownFunc(ctx)
}

// ShutdownCalls gets all the calls that were made to Shutdown.
// Check the length with:
//
//	len(mockedAPI.ShutdownCalls())
func (mock *APIMock) ShutdownCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}
