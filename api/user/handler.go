package user

type Handler struct {
	serv *Service
}

func NewUserHandler(serv *Service) *Handler {
	return &Handler{serv: serv}
}
