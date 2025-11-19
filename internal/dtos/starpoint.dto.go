package dtos

type Starpoint struct {
	ID                 string             `json:"id"`
	CummulativeAverage float64            `json:"cummulative_average"`
	TotalPoints        float64            `json:"total_points"`
	Programs           []StarpointProgram `json:"programs"`
}

type StarpointProgram struct {
	ID        string  `json:"id"`
	Session   string  `json:"session"`
	EventName string  `json:"event_name"`
	Type      string  `json:"type"`
	Level     string  `json:"level"`
	Points    float32 `json:"points"`
}
