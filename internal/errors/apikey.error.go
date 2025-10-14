package errors

var (
	ErrFailedToGenerateAPIKey = &CustomError{
		Message:    "Failed to generate API key",
		StatusCode: 500,
	}

	ErrInvalidAPIKey = &CustomError{
		Message:    "Invalid API key",
		StatusCode: 401,
	}

	ErrAPIKeyRequired = &CustomError{
		Message:    "API key is required",
		StatusCode: 401,
	}

	ErrFailedToEncryptWithAPIKey = &CustomError{
		Message:    "Failed to encrypt data with API key",
		StatusCode: 500,
	}

	ErrFailedToDecryptWithAPIKey = &CustomError{
		Message:    "Failed to decrypt data with API key",
		StatusCode: 500,
	}
)
