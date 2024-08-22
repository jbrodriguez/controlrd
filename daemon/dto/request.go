package dto

type Request struct {
	Params map[string]string `json:"params,omitempty"`
	Action string            `json:"action"`
}
