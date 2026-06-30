package constants

const (
	IiumPage                    = "https://www.iium.edu.my/"
	ImaluumPage                 = "https://imaluum.iium.edu.my/"
	ImaluumLogoutPage           = "https://imaluum.iium.edu.my/logout"
	ImaluumCasPage              = "https://cas.iium.edu.my:8448/cas/login?service=https%3a%2f%2fimaluum.iium.edu.my%2fhome"
	ImaluumCasLogoutPage        = "https://cas.iium.edu.my:8448/cas/logout?service=http://imaluum.iium.edu.my/"
	ImaluumProfilePage          = "https://imaluum.iium.edu.my/Profile"
	ImaluumLoginPage            = "https://cas.iium.edu.my:8448/cas/login?service=https%3a%2f%2fimaluum.iium.edu.my%2fhome?service=https%3a%2f%2fimaluum.iium.edu.my%2fhome"
	ImaluumHomePage             = "https://imaluum.iium.edu.my/home"
	ImaluumSchedulePage         = "https://imaluum.iium.edu.my/MyAcademic/schedule"
	ImaluumResultPage           = "https://imaluum.iium.edu.my/MyAcademic/result"
	ImaluumConfirmationSlipPage = "https://imaluum.iium.edu.my/MyAcademic/confirmation-sem"
	ImaluumExamSlipPage         = "https://imaluum.iium.edu.my/examslip"
	ImaluumFinalExamPage        = "https://imaluum.iium.edu.my/MyAcademic/final-exam"
	ImaluumDisciplinaryPage     = "https://imaluum.iium.edu.my/Disciplinary"
	ImaluumCarryMarkPage        = "https://imaluum.iium.edu.my/MyAcademic/cam"
	ImaluumStudyPlanPage        = "https://imaluum.iium.edu.my/MyAcademic/studyplan"
	ImaluumStarpointPage        = "https://imaluum.iium.edu.my/CoCurriculum"
	// DefaultUserAgent is a realistic browser User-Agent. i-Ma'luum's bot
	// protection (notably on the /MyAcademic/* paths) returns 403 Forbidden for
	// requests with a non-browser User-Agent, so all scrapers must send this.
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// DefaultAcceptHeader must contain text/html. i-Ma'luum's /MyAcademic/*
	// controllers return 403 Forbidden for requests whose Accept header does not
	// include text/html (even Accept: */* is rejected), so all scrapers must send
	// this. Plain colly sends no Accept header by default.
	DefaultAcceptHeader = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	TimeSeparator       = "-"
	DebugUserCookie     = "gomaluum_debug_user"
	DebugUsername       = "2214227"
	DebugPassword       = "fakepass"
)
