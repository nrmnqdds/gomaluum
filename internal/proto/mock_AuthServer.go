// Code generated by mockery v2.52.1. DO NOT EDIT.

package auth_proto

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockAuthServer is an autogenerated mock type for the AuthServer type
type MockAuthServer struct {
	mock.Mock
}

type MockAuthServer_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAuthServer) EXPECT() *MockAuthServer_Expecter {
	return &MockAuthServer_Expecter{mock: &_m.Mock}
}

// Login provides a mock function with given fields: _a0, _a1
func (_m *MockAuthServer) Login(_a0 context.Context, _a1 *LoginRequest) (*LoginResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Login")
	}

	var r0 *LoginResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *LoginRequest) (*LoginResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *LoginRequest) *LoginResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*LoginResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *LoginRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAuthServer_Login_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Login'
type MockAuthServer_Login_Call struct {
	*mock.Call
}

// Login is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *LoginRequest
func (_e *MockAuthServer_Expecter) Login(_a0 interface{}, _a1 interface{}) *MockAuthServer_Login_Call {
	return &MockAuthServer_Login_Call{Call: _e.mock.On("Login", _a0, _a1)}
}

func (_c *MockAuthServer_Login_Call) Run(run func(_a0 context.Context, _a1 *LoginRequest)) *MockAuthServer_Login_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*LoginRequest))
	})
	return _c
}

func (_c *MockAuthServer_Login_Call) Return(_a0 *LoginResponse, _a1 error) *MockAuthServer_Login_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAuthServer_Login_Call) RunAndReturn(run func(context.Context, *LoginRequest) (*LoginResponse, error)) *MockAuthServer_Login_Call {
	_c.Call.Return(run)
	return _c
}

// mustEmbedUnimplementedAuthServer provides a mock function with no fields
func (_m *MockAuthServer) mustEmbedUnimplementedAuthServer() {
	_m.Called()
}

// MockAuthServer_mustEmbedUnimplementedAuthServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'mustEmbedUnimplementedAuthServer'
type MockAuthServer_mustEmbedUnimplementedAuthServer_Call struct {
	*mock.Call
}

// mustEmbedUnimplementedAuthServer is a helper method to define mock.On call
func (_e *MockAuthServer_Expecter) mustEmbedUnimplementedAuthServer() *MockAuthServer_mustEmbedUnimplementedAuthServer_Call {
	return &MockAuthServer_mustEmbedUnimplementedAuthServer_Call{Call: _e.mock.On("mustEmbedUnimplementedAuthServer")}
}

func (_c *MockAuthServer_mustEmbedUnimplementedAuthServer_Call) Run(run func()) *MockAuthServer_mustEmbedUnimplementedAuthServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAuthServer_mustEmbedUnimplementedAuthServer_Call) Return() *MockAuthServer_mustEmbedUnimplementedAuthServer_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockAuthServer_mustEmbedUnimplementedAuthServer_Call) RunAndReturn(run func()) *MockAuthServer_mustEmbedUnimplementedAuthServer_Call {
	_c.Run(run)
	return _c
}

// NewMockAuthServer creates a new instance of MockAuthServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAuthServer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAuthServer {
	mock := &MockAuthServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
