package dtos

type StudentBatch struct {
	Batch int `json:"batch"`
	Count int `json:"count"`
}

type LevelSummary struct {
	Level    string         `json:"level"`
	Students []StudentBatch `json:"students"`
}
