package dtos

type RequestSubscriptionType struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	User string `json:"user"`
	Tag  string `json:"tag"`
}
