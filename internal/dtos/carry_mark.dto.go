package dtos

type CarryMark struct {
	ID       string             `json:"id"`
	Session  string             `json:"session"`
	Subjects []CarryMarkSubject `json:"subjects"`
}

type CarryMarkSubject struct {
	ID             string               `json:"id"`
	Code           string               `json:"code"`
	Section        string               `json:"section"`
	Course         string               `json:"course"`
	CreditHour     string               `json:"credit_hour"`
	TotalCarryMark string               `json:"total_carry_mark"`
	Components     []CarryMarkComponent `json:"components"`
}

type CarryMarkComponent struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MarkingScore string `json:"marking_score"`
	ActualScore  string `json:"actual_score"`
}
