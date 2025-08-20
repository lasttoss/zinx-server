package models

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
)

type UserModel struct {
	Id          bson.ObjectID `bson:"_id"`
	UserId      string        `bson:"user_id"`
	GoogleId    string        `bson:"google_id"`
	AppleId     string        `bson:"apple_id"`
	FacebookId  string        `bson:"facebook_id"`
	DeviceId    string        `bson:"device_id"`
	DisplayName string        `bson:"display_name"`
	AvatarUrl   string        `bson:"avatar_url"`
	CreatedAt   time.Time     `bson:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"`
}

func NewDeviceIdUserModel(deviceId string) *UserModel {
	return &UserModel{
		Id:          bson.NewObjectID(),
		UserId:      uuid.NewString(),
		DeviceId:    deviceId,
		GoogleId:    "",
		AppleId:     "",
		FacebookId:  "",
		DisplayName: "",
		AvatarUrl:   "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func NewGoogleIdUserModel(googleId string) *UserModel {
	return &UserModel{
		Id:          bson.NewObjectID(),
		GoogleId:    googleId,
		AppleId:     "",
		FacebookId:  "",
		DeviceId:    "",
		DisplayName: "",
		AvatarUrl:   "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func NewAppleIdUserModel(AppleId string) *UserModel {
	return &UserModel{
		Id:          bson.NewObjectID(),
		AppleId:     AppleId,
		GoogleId:    "",
		FacebookId:  "",
		DeviceId:    "",
		DisplayName: "",
		AvatarUrl:   "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
