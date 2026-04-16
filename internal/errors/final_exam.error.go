package errors

var ErrNoFinalExam = &CustomError{
	Message:    "User has no final exam timetable",
	StatusCode: 404,
}
