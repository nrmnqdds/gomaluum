package errors

var ErrScheduleIsEmpty = &CustomError{
	Message:    "Schedule is empty",
	StatusCode: 404,
}
