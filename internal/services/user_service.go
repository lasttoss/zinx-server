package services

import (
	"encoding/json"
	"zinx-server/internal/mappers"
	"zinx-server/internal/repositories"
	"zinx-server/internal/utils"
)

type UserService interface {
	GetUserByUserId(userId string) ([]byte, []byte)
}

type userService struct {
	userRepository repositories.UserRepository
}

func (u userService) GetUserByUserId(userId string) ([]byte, []byte) {
	item, repoErr := u.userRepository.FindByUserId(userId)
	if repoErr != nil {
		return nil, utils.NewApiError(utils.SystemError)
	}
	if item == nil {
		return nil, utils.NewApiError(utils.ItemNotFoundError)
	}
	response := mappers.NewUserResponse(*item)
	responseBytes, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		return nil, utils.NewApiError(utils.EncodeJsonError)
	}
	return responseBytes, nil
}

func NewUserService(userRepository repositories.UserRepository) UserService {
	return userService{
		userRepository: userRepository,
	}
}
