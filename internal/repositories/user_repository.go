package repositories

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"zinx-server/internal/models"
)

type UserRepository interface {
	FindByDeviceId(deviceId string) (*models.UserModel, error)
	FindByUserId(userId string) (*models.UserModel, error)
	FindByGoogleId(googleId string) (*models.UserModel, error)
	FindByAppleId(appleId string) (*models.UserModel, error)
	Save(user *models.UserModel) error
	Update(id bson.ObjectID, update bson.M) error
}

type userRepository struct {
	userCollection *mongo.Collection
}

func (u userRepository) FindByUserId(userId string) (*models.UserModel, error) {
	var item *models.UserModel
	err := u.userCollection.FindOne(context.TODO(), bson.M{"user_id": userId}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (u userRepository) Save(user *models.UserModel) error {
	_, err := u.userCollection.InsertOne(context.TODO(), user)
	if err != nil {
		return err
	}
	return nil
}

func (u userRepository) Update(id bson.ObjectID, update bson.M) error {
	_, err := u.userCollection.UpdateByID(context.TODO(), id, update)
	if err != nil {
		return err
	}
	return nil
}

func (u userRepository) FindByDeviceId(deviceId string) (*models.UserModel, error) {
	var item *models.UserModel
	err := u.userCollection.FindOne(context.TODO(), bson.M{"device_id": deviceId}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (u userRepository) FindByGoogleId(googleId string) (*models.UserModel, error) {
	var item *models.UserModel
	err := u.userCollection.FindOne(context.TODO(), bson.M{"google_id": googleId}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (u userRepository) FindByAppleId(appleId string) (*models.UserModel, error) {
	var item *models.UserModel
	err := u.userCollection.FindOne(context.TODO(), bson.M{"apple_id": appleId}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func NewUserRepository(userCollection *mongo.Collection) UserRepository {
	return &userRepository{
		userCollection: userCollection,
	}
}
