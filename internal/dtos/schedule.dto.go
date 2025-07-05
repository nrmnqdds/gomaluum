package dtos

type WeekTime struct {
	Start     string `json:"start"`
	StartUnix int64  `json:"start_unix,omitempty"`
	End       string `json:"end"`
	EndUnix   int64  `json:"end_unix,omitempty"`
	Day       uint8  `json:"day"`
}

type ScheduleSubject struct {
	ID         string     `json:"id"`
	CourseCode string     `json:"course_code"`
	CourseName string     `json:"course_name"`
	Venue      string     `json:"venue"`
	Lecturer   string     `json:"lecturer"`
	Timestamps []WeekTime `json:"timestamps"`
	Chr        float64    `json:"chr"`
	Section    uint32     `json:"section"`
}

type ScheduleResponse struct {
	ID           string            `json:"id"`
	SessionName  string            `json:"session_name"`
	SessionQuery string            `json:"session_query"`
	Schedule     []ScheduleSubject `json:"schedule"`
}
