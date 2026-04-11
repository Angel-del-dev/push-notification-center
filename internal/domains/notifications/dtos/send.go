package dtos

type RequestSendType struct {
	User    string `json:"user"`
	Tag     string `json:"tag"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Icon    string `json:"icon"`
}
