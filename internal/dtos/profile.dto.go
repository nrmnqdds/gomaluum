package dtos

type Profile struct {
	ImageURL      string `json:"image_url"`
	Name          string `json:"name"`
	MatricNo      string `json:"matric_no"`
	Level         string `json:"level"`
	Kuliyyah      string `json:"kuliyyah"`
	IC            string `json:"ic"`
	Gender        string `json:"gender"`
	Birthday      string `json:"birthday"`
	Religion      string `json:"religion"`
	MaritalStatus string `json:"marital_status"`
	Address       string `json:"address"`
}
