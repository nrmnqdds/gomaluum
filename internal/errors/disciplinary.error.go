package errors

var ErrNoDisciplinaryRecord = &CustomError{
	Message:    "User has no disciplinary or compound records",
	StatusCode: 404,
}
