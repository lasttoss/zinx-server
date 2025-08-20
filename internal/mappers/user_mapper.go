package mappers

import "zinx-server/internal/models"

type UserResponse struct {
	Id          string `json:"id"`
	UserId      string `json:"userId"`
	DeviceId    string `json:"deviceId"`
	GoogleId    string `json:"googleId"`
	FacebookId  string `json:"facebookId"`
	AppleId     string `json:"appleId"`
	DisplayName string `json:"displayName"`
	AvatarUrl   string `json:"avatarUrl"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func NewUserResponse(item models.UserModel) UserResponse {
	return UserResponse{
		Id:          item.Id.Hex(),
		UserId:      item.UserId,
		DeviceId:    item.DeviceId,
		GoogleId:    item.GoogleId,
		AppleId:     item.AppleId,
		DisplayName: item.DisplayName,
		AvatarUrl:   item.AvatarUrl,
		CreatedAt:   item.CreatedAt.String(),
		UpdatedAt:   item.UpdatedAt.String(),
	}
}
