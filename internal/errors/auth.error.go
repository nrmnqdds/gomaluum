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

	ErrCookieJarCreationFailed = &CustomError{
		Message:    "Failed to create cookie jar",
		StatusCode: 500,
	}

	ErrEncryptionFailed = &CustomError{
		Message:    "Failed to encrypt password",
		StatusCode: 500,
	}

	ErrFailedToCloseRequestBody = &CustomError{
		Message:    "Failed to close request body",
		StatusCode: 500,
	}

	ErrFailedToCloseResponseBody = &CustomError{
		Message:    "Failed to close response body",
		StatusCode: 500,
	}
)
