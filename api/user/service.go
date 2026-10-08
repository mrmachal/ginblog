package user

type Service struct {
	repo Repository
}

func NewUserService(repo Repository) *Service {
	return &Service{repo: repo}
}
