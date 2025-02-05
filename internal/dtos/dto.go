package dtos

type ResponseDTO struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}
