package db

type ResearchSession struct {
	ID              string `json:"id"`
	CodebaseID      string `json:"codebaseId"`
	CodebasePath    string `json:"codebasePath"`
	CodebaseName    string `json:"codebaseName"`
	CodebaseVersion string `json:"codebaseVersion"`
	Title           string `json:"title"`
	State           string `json:"state"`
	CreatedAt       int64  `json:"createdAt"`
	ArchivedAt      *int64 `json:"archivedAt,omitempty"`
}

type ResearchReport struct {
	ID         string `json:"id"`
	SessionID  string `json:"sessionId"`
	TurnID     string `json:"turnId"`
	StreamData string `json:"streamData"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
}

type SavedReport struct {
	ID           string `json:"id"`
	SessionID    string `json:"sessionId"`
	CodebaseID   string `json:"codebaseId"`
	Title        string `json:"title"`
	Query        string `json:"query"`
	Content      string `json:"content"`
	CodebaseName string `json:"codebaseName"`
	CodebasePath string `json:"codebasePath"`
	CreatedAt    int64  `json:"createdAt"`
}
