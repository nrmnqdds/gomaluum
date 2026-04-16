package dtos

type Disciplinary struct {
	ID        string                `json:"id"`
	Compounds []DisciplinaryCompound `json:"compounds"`
}

type DisciplinaryCompound struct {
	ID          string                   `json:"id"`
	Session     string                   `json:"session"`
	OffenceDate string                   `json:"offence_date"`
	CompoundNo  string                   `json:"compound_no"`
	Description string                   `json:"description"`
	Agency      string                   `json:"agency"`
	Status      string                   `json:"status"`
	Fine        string                   `json:"fine"`
	DueDate     string                   `json:"due_date"`
	Links       []DisciplinaryCompoundLink `json:"links"`
}

type DisciplinaryCompoundLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
