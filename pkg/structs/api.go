// structs for api request and response
package structs

// AUTH
type RegisterUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type RegisterUserResponse struct {
	Ok      bool   `json:"ok"`
	UserID  int64  `json:"user_id"`
	Message string `json:"message"`
}

type RequestVerifyEmail struct {
	UserID int64  `json:"user_id"`
	OTP    string `json:"otp"`
}

type AuthenticateResponse struct {
	Ok           bool   `json:"ok"`
	Message      string `json:"message"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type RequestLogin struct {
	GrantType string `json:"grant_type"` // base on grant type, we need either password or otp
	Email     string `json:"email"`
	Password  string `json:"password"`
	OTP       string `json:"otp"`
}

type RequestGetOTPLogin struct {
	Email string `json:"email"`
}

type RequestEditUser struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type ReponeGetUser struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
}

// PROBLEMS
type RequestCreateProblem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ContestID   int64  `json:"contest_id"`
	IsPrivate   bool
	Hardness    int64 `json:"hardness"`
}

type ResponseCreateProblem struct {
	ProblemID int64 `json:"problem_Id"`
}

type RequestListProblems struct {
	OrderedBy  string `json:"ordered_by"`
	Descending bool   `json:"descending"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	GetCount   bool   `json:"get_count"`
}

type ResponseListProblems struct {
	TotalCount int                        `json:"total_count,omitempty"`
	Problems   []ResponseListProblemsItem `json:"problems"`
}

type ResponseListProblemsItem struct {
	ProblemID  int64  `json:"problem_id"`
	Title      string `json:"title"`
	SolveCount int64  `json:"solve_count"`
	Hardness   int64  `json:"hardness"`
}

type ResponseGetProblem struct {
	ProblemID   int64  `json:"problem_Id"`
	Title       string `json:"title"`
	SolveCount  int64  `json:"solve_count"`
	Hardness    int64  `json:"hardness"`
	Description string `json:"description"`
	IsOwned     bool   `json:"is_owned"`
}

type RequestUpdateProblem struct {
	Id          int64
	Title       string `json:"title"`
	Description string `json:"description"`
	Hardness    int64  `json:"hardness"`
}

// SUBMISSIONS
type RequestSubmit struct {
	UserID      int64
	ProblemID   int64
	ContestID   int64
	Code        []byte
	FileName    string
	ContentType string
	Language    string
}

type ResponseGetSubmission struct {
	Metadata SubmissionMetadata
	RawCode  []byte `json:"data"`
}

type ResponseGetSubmissionResults struct {
	Verdicts       []Verdict `json:"verdicts"`
	ServiceMessage string    `json:"service_message"`
	TestCaseID     int64     `json:"testcase_id"`
	ErrorMessage   string    `json:"error_message"`
}

type RequestListSubmissions struct {
	ProblemID  int64 `json:"problem_id"`
	UserID     int64 `json:"user_id"`
	ContestID  int64 `json:"contest_id,omitempty"`
	Descending bool  `json:"descending"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	GetCount   bool  `json:"get_count"`
}

type ResponseListSubmissions struct {
	TotalCount  int                           `json:"total_count,omitempty"`
	Submissions []ResponseListSubmissionsItem `json:"submissions"`
}

type SubmissionListMetadata struct {
	ID           int64  `json:"submission_id"`
	UserID       int64  `json:"user_id,omitempty"`
	Language     string `json:"language"`
	CreatedAt    string `json:"created_at"`
	FileName     string `json:"file_name"`
	Score        int    `json:"int"`
	ProblemID    int64  `json:"problem_id"`
	ProblemTitle string `json:"problem_title"`
}

type ResponseListSubmissionsItem struct {
	Metadata SubmissionListMetadata       `json:"metadata"`
	Results  ResponseGetSubmissionResults `json:"results"`
}

// CONTESTS
type RequestCreateContest struct {
	Title     string `json:"title"`
	StartTime int64  `json:"start_time"`
	Duration  int    `json:"duration"`
}

type ResponseCreateContest struct {
	ContestID int64 `json:"contest_Id"`
}

type ResponseGetContest struct {
	ContestID      int64              `json:"contest_Id"`
	Title          string             `json:"title"`
	Problems       []ContestProblem   `json:"problems"`
	StartTime      int64              `json:"start_time"`
	Duration       int                `json:"duration"`
	RegisterStatus RegistrationStatus `json:"register_status,omitempty"`
}

type RequestListContests struct {
	UserID       int64 `json:"user_id"`
	Descending   bool  `json:"descending"`
	Limit        int   `json:"limit"`
	Offset       int   `json:"offset"`
	MyContest    bool  `json:"my_contest"`
	Started      bool  `json:"started"`
	OwnedContest bool  `json:"owned_contest"`
	GetCount     bool  `json:"get_count"`
}

type RequestUpdateContest struct {
	Title     string `json:"title"`
	StartTime int64  `json:"start_time"`
	Duration  int    `json:"duration"`
}

type ResponseListContestsItem struct {
	ContestID      int64              `json:"contest_Id"`
	Title          string             `json:"title"`
	RegisterStatus RegistrationStatus `json:"register_status,omitempty"`
}

type ResponseListContests struct {
	TotalCount int                        `json:"total_count,omitempty"`
	Contests   []ResponseListContestsItem `json:"contests"`
}

type RequestAddProblemContest struct {
	ContestID int64 `json:"contest_Id"`
	ProblemID int64 `json:"problem_Id"`
}

type RequestGetScoreboard struct {
	ContestID int64
	GetCount  bool
	Limit     int
	Offset    int
}

type RequestRemoveProblemContest struct {
	ContestID int64 `json:"contest_Id"`
	ProblemID int64 `json:"problem_Id"`
}

type ScoreboardProblem struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type ScoreboardUserStanding struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"user_name"`
	Scores   []int  `json:"scores"`
}
type ResponseGetContestScoreboard struct {
	Count    int                      `json:"count,omitempty"`
	Users    []ScoreboardUserStanding `json:"users"`
	Problems []ScoreboardProblem      `json:"problems"`
}

type ResponseGetTestcase struct {
	ID     int64  `json:"id"`
	Input  string `json:"input"`
	Output string `json:"output"`
}
