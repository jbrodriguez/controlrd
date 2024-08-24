package dto

type Origin struct {
	Name      string `json:"name"`
	Protocol  string `json:"protocol"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	Address   string `json:"address"`
	ErrorCode string `json:"errorCode"`
	ErrorText string `json:"errorText"`
}
