package errors

var ErrNoCarryMark = &CustomError{
	Message:    "User has no carry mark data",
	StatusCode: 404,
}
