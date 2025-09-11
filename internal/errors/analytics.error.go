package errors

var (
	ErrFailedToQueryDB = &CustomError{
		Message:    "Failed to query database",
		StatusCode: 500,
	}

	ErrFailedToMapDBRows = &CustomError{
		Message:    "Failed to map database rows",
		StatusCode: 500,
	}
)
