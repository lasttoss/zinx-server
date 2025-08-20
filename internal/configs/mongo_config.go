package configs

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"log"
	"time"
	"zinx-server/internal/constants"
)

var MongoDB struct {
	UserCollection *mongo.Collection
}

func ConnectMongo() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(ServerConfig.Database.Url))
	if err != nil {
		log.Fatalf("Connect MongoDB Error: %v", err)
	}

	//defer func() {
	//	if err = client.Disconnect(ctx); err != nil {
	//		log.Fatalf("Disconnect MongoDB Error: %v", err)
	//	}
	//}()

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("Connect MongoDB Error: %v", err)
	}

	database := client.Database(ServerConfig.Database.Name)
	log.Println("[DB] connect successfully")

	userCollection := database.Collection(constants.UserCollectionName)
	err = createMongoIndex(client, constants.UserCollectionName, bson.D{{Key: "user_id", Value: -1}})
	if err != nil {
		log.Fatalf("Create User Index Error: %v", err)
	}
	err = createMongoIndex(client, constants.UserCollectionName, bson.D{{Key: "device_id", Value: -1}})
	if err != nil {
		log.Fatalf("Create User Index Error: %v", err)
	}
	err = createMongoIndex(client, constants.UserCollectionName, bson.D{{Key: "google_id", Value: -1}})
	if err != nil {
		log.Fatalf("Create User Index Error: %v", err)
	}
	err = createMongoIndex(client, constants.UserCollectionName, bson.D{{Key: "apple_id", Value: -1}})
	if err != nil {
		log.Fatalf("Create User Index Error: %v", err)
	}
	MongoDB.UserCollection = userCollection

}

func createMongoUnique(client *mongo.Client, collectionName string, params bson.D) error {
	collection := client.Database(ServerConfig.Database.Name).Collection(collectionName)
	compoundUniqueIndexModel := mongo.IndexModel{
		Keys:    params,
		Options: options.Index().SetUnique(true),
	}
	_, err := collection.Indexes().CreateMany(context.TODO(), []mongo.IndexModel{compoundUniqueIndexModel})
	if err != nil {
		return err
	}
	return nil
}

func createMongoIndex(client *mongo.Client, collectionName string, params bson.D) error {
	collection := client.Database(ServerConfig.Database.Name).Collection(collectionName)

	indexModel := mongo.IndexModel{
		Keys: params,
	}
	opts := options.CreateIndexes()

	_, err := collection.Indexes().CreateOne(context.TODO(), indexModel, opts)
	if err != nil {
		return err
	}

	return nil
}
