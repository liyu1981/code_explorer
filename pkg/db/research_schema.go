package db

type ResearchSession struct {
	ID           string `json:"id"`
	CodebaseID   string `json:"codebaseId"`
	CodebasePath string `json:"codebasePath"`
	Title        string `json:"title"`
	State        string `json:"state"`
	CreatedAt    int64  `json:"createdAt"`
	ArchivedAt   *int64 `json:"archivedAt,omitempty"`
}

type ResearchReport struct {
	ID         string `json:"id"`
	SessionID  string `json:"sessionId"`
	TurnID     string `json:"turnId"`
	StreamData string `json:"streamData"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
}
