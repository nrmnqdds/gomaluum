package errors

var (
	ErrFailedToGeneratePASETO = &CustomError{
		Message:    "Failed to generate PASETO token",
		StatusCode: 500,
	}

	ErrFailedToDecodePASETO = &CustomError{
		Message:    "Failed to decode PASETO token",
		StatusCode: 500,
	}

	ErrInvalidPASETOIssuer = &CustomError{
		Message:    "Invalid PASETO issuer",
		StatusCode: 401,
	}

	ErrFailedToCreatePASETOPublicKey = &CustomError{
		Message:    "Failed to create PASETO public key",
		StatusCode: 500,
	}

	ErrFailedToCreatePASETOPrivateKey = &CustomError{
		Message:    "Failed to create PASETO private key",
		StatusCode: 500,
	}
)
