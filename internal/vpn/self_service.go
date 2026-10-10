package vpn

type SelfReply struct {
	ID       string `json:"id"`
	Accepted bool   `json:"accepted"`
	Detail   string `json:"detail"`
}
