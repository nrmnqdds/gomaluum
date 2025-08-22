package errors

var ErrNoStarpoint = &CustomError{
	Message:    "User has no starpoint",
	StatusCode: 404,
}
