// Code generated by mockery v2.52.1. DO NOT EDIT.

package server

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"
)

// MockHandlers is an autogenerated mock type for the Handlers type
type MockHandlers struct {
	mock.Mock
}

type MockHandlers_Expecter struct {
	mock *mock.Mock
}

func (_m *MockHandlers) EXPECT() *MockHandlers_Expecter {
	return &MockHandlers_Expecter{mock: &_m.Mock}
}

// LoginHandler provides a mock function with given fields: w, r
func (_m *MockHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	_m.Called(w, r)
}

// MockHandlers_LoginHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'LoginHandler'
type MockHandlers_LoginHandler_Call struct {
	*mock.Call
}

// LoginHandler is a helper method to define mock.On call
//   - w http.ResponseWriter
//   - r *http.Request
func (_e *MockHandlers_Expecter) LoginHandler(w interface{}, r interface{}) *MockHandlers_LoginHandler_Call {
	return &MockHandlers_LoginHandler_Call{Call: _e.mock.On("LoginHandler", w, r)}
}

func (_c *MockHandlers_LoginHandler_Call) Run(run func(w http.ResponseWriter, r *http.Request)) *MockHandlers_LoginHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(http.ResponseWriter), args[1].(*http.Request))
	})
	return _c
}

func (_c *MockHandlers_LoginHandler_Call) Return() *MockHandlers_LoginHandler_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockHandlers_LoginHandler_Call) RunAndReturn(run func(http.ResponseWriter, *http.Request)) *MockHandlers_LoginHandler_Call {
	_c.Run(run)
	return _c
}

// LogoutHandler provides a mock function with given fields: w, r
func (_m *MockHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	_m.Called(w, r)
}

// MockHandlers_LogoutHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'LogoutHandler'
type MockHandlers_LogoutHandler_Call struct {
	*mock.Call
}

// LogoutHandler is a helper method to define mock.On call
//   - w http.ResponseWriter
//   - r *http.Request
func (_e *MockHandlers_Expecter) LogoutHandler(w interface{}, r interface{}) *MockHandlers_LogoutHandler_Call {
	return &MockHandlers_LogoutHandler_Call{Call: _e.mock.On("LogoutHandler", w, r)}
}

func (_c *MockHandlers_LogoutHandler_Call) Run(run func(w http.ResponseWriter, r *http.Request)) *MockHandlers_LogoutHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(http.ResponseWriter), args[1].(*http.Request))
	})
	return _c
}

func (_c *MockHandlers_LogoutHandler_Call) Return() *MockHandlers_LogoutHandler_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockHandlers_LogoutHandler_Call) RunAndReturn(run func(http.ResponseWriter, *http.Request)) *MockHandlers_LogoutHandler_Call {
	_c.Run(run)
	return _c
}

// ProfileHandler provides a mock function with given fields: w, r
func (_m *MockHandlers) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	_m.Called(w, r)
}

// MockHandlers_ProfileHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ProfileHandler'
type MockHandlers_ProfileHandler_Call struct {
	*mock.Call
}

// ProfileHandler is a helper method to define mock.On call
//   - w http.ResponseWriter
//   - r *http.Request
func (_e *MockHandlers_Expecter) ProfileHandler(w interface{}, r interface{}) *MockHandlers_ProfileHandler_Call {
	return &MockHandlers_ProfileHandler_Call{Call: _e.mock.On("ProfileHandler", w, r)}
}

func (_c *MockHandlers_ProfileHandler_Call) Run(run func(w http.ResponseWriter, r *http.Request)) *MockHandlers_ProfileHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(http.ResponseWriter), args[1].(*http.Request))
	})
	return _c
}

func (_c *MockHandlers_ProfileHandler_Call) Return() *MockHandlers_ProfileHandler_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockHandlers_ProfileHandler_Call) RunAndReturn(run func(http.ResponseWriter, *http.Request)) *MockHandlers_ProfileHandler_Call {
	_c.Run(run)
	return _c
}

// ResultHandler provides a mock function with given fields: w, r
func (_m *MockHandlers) ResultHandler(w http.ResponseWriter, r *http.Request) {
	_m.Called(w, r)
}

// MockHandlers_ResultHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ResultHandler'
type MockHandlers_ResultHandler_Call struct {
	*mock.Call
}

// ResultHandler is a helper method to define mock.On call
//   - w http.ResponseWriter
//   - r *http.Request
func (_e *MockHandlers_Expecter) ResultHandler(w interface{}, r interface{}) *MockHandlers_ResultHandler_Call {
	return &MockHandlers_ResultHandler_Call{Call: _e.mock.On("ResultHandler", w, r)}
}

func (_c *MockHandlers_ResultHandler_Call) Run(run func(w http.ResponseWriter, r *http.Request)) *MockHandlers_ResultHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(http.ResponseWriter), args[1].(*http.Request))
	})
	return _c
}

func (_c *MockHandlers_ResultHandler_Call) Return() *MockHandlers_ResultHandler_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockHandlers_ResultHandler_Call) RunAndReturn(run func(http.ResponseWriter, *http.Request)) *MockHandlers_ResultHandler_Call {
	_c.Run(run)
	return _c
}

// ScheduleHandler provides a mock function with given fields: w, r
func (_m *MockHandlers) ScheduleHandler(w http.ResponseWriter, r *http.Request) {
	_m.Called(w, r)
}

// MockHandlers_ScheduleHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ScheduleHandler'
type MockHandlers_ScheduleHandler_Call struct {
	*mock.Call
}

// ScheduleHandler is a helper method to define mock.On call
//   - w http.ResponseWriter
//   - r *http.Request
func (_e *MockHandlers_Expecter) ScheduleHandler(w interface{}, r interface{}) *MockHandlers_ScheduleHandler_Call {
	return &MockHandlers_ScheduleHandler_Call{Call: _e.mock.On("ScheduleHandler", w, r)}
}

func (_c *MockHandlers_ScheduleHandler_Call) Run(run func(w http.ResponseWriter, r *http.Request)) *MockHandlers_ScheduleHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(http.ResponseWriter), args[1].(*http.Request))
	})
	return _c
}

func (_c *MockHandlers_ScheduleHandler_Call) Return() *MockHandlers_ScheduleHandler_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockHandlers_ScheduleHandler_Call) RunAndReturn(run func(http.ResponseWriter, *http.Request)) *MockHandlers_ScheduleHandler_Call {
	_c.Run(run)
	return _c
}

// NewMockHandlers creates a new instance of MockHandlers. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockHandlers(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockHandlers {
	mock := &MockHandlers{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
