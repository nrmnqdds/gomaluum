package dtos

type ResponseDTO struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
}
