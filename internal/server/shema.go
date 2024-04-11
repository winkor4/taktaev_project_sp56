package server

type registerSchema struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
