package dto

type Response struct {
	Message string   `json:"message"`
	Logs    []string `json:"logs,omitempty"`
	Error   string   `json:"error,omitempty"`
}
