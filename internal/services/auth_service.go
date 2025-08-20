package services

import (
	"encoding/json"
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
	"zinx-server/internal/constants"
	"zinx-server/internal/mappers"
	"zinx-server/internal/models"
	"zinx-server/internal/repositories"
	"zinx-server/internal/utils"
)

type AuthService interface {
	AuthByToken(request mappers.AuthRequest) (string, []byte, []byte)
	AuthByDevice(request mappers.AuthRequest) (string, []byte, []byte)
	AuthByGoogle(request mappers.AuthRequest) (string, []byte, []byte)
	AuthByApple(request mappers.AuthRequest) (string, []byte, []byte)
}

type authService struct {
	userRepository      repositories.UserRepository
	googleClientId      string
	appleGoogleClientId string
	appleClientId       string
}

func (a authService) AuthByToken(request mappers.AuthRequest) (string, []byte, []byte) {
	claims, jwtErr := utils.ValidateJWT(request.Id)
	if jwtErr != nil {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}
	userId, ok := claims["sub"].(string)
	if !ok {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}

	item, repoErr := a.userRepository.FindByUserId(userId)
	if repoErr != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}
	if item == nil {
		return "", nil, utils.NewApiError(utils.ItemNotFoundError)
	}

	item.UpdatedAt = time.Now()
	update := bson.M{"$set": bson.M{"updated_at": item.UpdatedAt}}
	repoErr = a.userRepository.Update(item.Id, update)
	if repoErr != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}
	data := mappers.NewUserResponse(*item)
	token, jwtErr := utils.GenerateJWT(item, time.Hour*24*30)
	if jwtErr != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}
	response := mappers.NewAuthResponse(token, data)
	responseStr, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		return "", nil, utils.NewApiError(utils.EncodeJsonError)
	}

	return item.UserId, responseStr, nil
}

func (a authService) AuthByDevice(request mappers.AuthRequest) (string, []byte, []byte) {
	item, repoErr := a.userRepository.FindByDeviceId(request.Id)
	if repoErr != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}
	if item == nil {
		item = models.NewDeviceIdUserModel(request.Id)
		repoErr := a.userRepository.Save(item)
		if repoErr != nil {
			return "", nil, utils.NewApiError(utils.SystemError)
		}
	} else {
		item.UpdatedAt = time.Now()
		update := bson.M{"$set": bson.M{"updated_at": item.UpdatedAt}}
		repoErr := a.userRepository.Update(item.Id, update)
		if repoErr != nil {
			return "", nil, utils.NewApiError(utils.SystemError)
		}
	}
	data := mappers.NewUserResponse(*item)
	token, jwtErr := utils.GenerateJWT(item, time.Hour*24*30)
	if jwtErr != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}
	response := mappers.NewAuthResponse(token, data)
	responseStr, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		return "", nil, utils.NewApiError(utils.EncodeJsonError)
	}

	return item.UserId, responseStr, nil
}

func (a authService) AuthByGoogle(request mappers.AuthRequest) (string, []byte, []byte) {
	claims, jwtErr := utils.ExtractClaims(request.Id)
	if jwtErr != nil {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}

	aud, ok := claims[constants.JwtAud].(string)
	if !ok {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}
	if !(aud != a.googleClientId || aud != a.appleGoogleClientId) {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}

	googleId, ok := claims[constants.JwtSub].(string)
	if !ok {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}
	item, err := a.userRepository.FindByGoogleId(googleId)
	if err != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}

	if item == nil {
		item = models.NewGoogleIdUserModel(googleId)
		repoErr := a.userRepository.Save(item)
		if repoErr != nil {
			return "", nil, utils.NewApiError(utils.SystemError)
		}
	} else {
		item.UpdatedAt = time.Now()
		update := bson.M{"$set": bson.M{"updated_at": item.UpdatedAt}}
		repoErr := a.userRepository.Update(item.Id, update)
		if repoErr != nil {
			return "", nil, utils.NewApiError(utils.SystemError)
		}
	}

	data := mappers.NewUserResponse(*item)
	token, jwtErr := utils.GenerateJWT(item, time.Hour*24*30)
	if jwtErr != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}
	response := mappers.NewAuthResponse(token, data)
	responseStr, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		return "", nil, utils.NewApiError(utils.EncodeJsonError)
	}
	return item.UserId, responseStr, nil
}

func (a authService) AuthByApple(request mappers.AuthRequest) (string, []byte, []byte) {
	claims, jwtErr := utils.ExtractClaims(request.Id)
	if jwtErr != nil {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}

	aud, ok := claims[constants.JwtAud].(string)
	if !ok {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}
	if aud != a.appleClientId {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}

	appleId, ok := claims[constants.JwtSub].(string)
	if !ok {
		return "", nil, utils.NewApiError(utils.JwtClaimsError)
	}
	item, err := a.userRepository.FindByAppleId(appleId)
	if err != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}

	if item == nil {
		item = models.NewAppleIdUserModel(appleId)
		repoErr := a.userRepository.Save(item)
		if repoErr != nil {
			return "", nil, utils.NewApiError(utils.SystemError)
		}
	} else {
		item.UpdatedAt = time.Now()
		update := bson.M{"$set": bson.M{"updated_at": item.UpdatedAt}}
		repoErr := a.userRepository.Update(item.Id, update)
		if repoErr != nil {
			return "", nil, utils.NewApiError(utils.SystemError)
		}
	}

	data := mappers.NewUserResponse(*item)
	token, jwtErr := utils.GenerateJWT(item, time.Hour*24*30)
	if jwtErr != nil {
		return "", nil, utils.NewApiError(utils.SystemError)
	}
	response := mappers.NewAuthResponse(token, data)
	responseStr, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		return "", nil, utils.NewApiError(utils.EncodeJsonError)
	}
	return item.UserId, responseStr, nil
}

func NewAuthService(userRepository repositories.UserRepository, googleClientId, appleGoogleClientId, appleClientId string) AuthService {
	return &authService{
		userRepository:      userRepository,
		googleClientId:      googleClientId,
		appleGoogleClientId: appleGoogleClientId,
		appleClientId:       appleClientId,
	}
}
