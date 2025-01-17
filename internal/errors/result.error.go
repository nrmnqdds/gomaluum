package errors

var ErrResultIsEmpty = &CustomError{
	Message:    "Result is empty",
	StatusCode: 500,
}
