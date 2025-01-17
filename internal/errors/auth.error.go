package errors

var (
	ErrLoginFailed = &CustomError{
		Message:    "Username or password is incorrect",
		StatusCode: 401,
	}

	ErrURLParseFailed = &CustomError{
		Message:    "Failed to parse URL",
		StatusCode: 500,
	}
)
