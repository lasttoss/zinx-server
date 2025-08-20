package mappers

type AuthRequest struct {
	Id string `json:"id" validate:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	Data  UserResponse `json:"data"`
}

func NewAuthResponse(token string, item UserResponse) AuthResponse {
	return AuthResponse{
		Token: token,
		Data:  item,
	}
}
