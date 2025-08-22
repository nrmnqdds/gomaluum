package errors

var ErrDownloadFailed = &CustomError{
	Message:    "Failed to download the file",
	StatusCode: 500,
}
