package dtos

type FinalExam struct {
	ID    string          `json:"id"`
	Exams []FinalExamItem `json:"exams"`
}

type FinalExamItem struct {
	ID             string `json:"id"`
	SubjectCode    string `json:"subject_code"`
	SubjectName    string `json:"subject_name"`
	SubjectSection string `json:"subject_section"`
	Date           string `json:"date"`
	Time           string `json:"time"`
	Venue          string `json:"venue"`
	Seat           string `json:"seat"`
}
