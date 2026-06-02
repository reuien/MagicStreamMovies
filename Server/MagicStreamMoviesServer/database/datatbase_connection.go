package database

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var Client *mongo.Client
func Connect() *mongo.Client {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning unable to find .env file")
	}

	MongoDB := os.Getenv("MONGODB_URI")
	if MongoDB == "" {
		log.Fatal("MONGODB_URI not set !")
	}
	fmt.Println("MongoDB URI : ", MongoDB)

	clientOptions := options.Client().ApplyURI(MongoDB)
	// connect to our database
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal("fucking mongoDB connection failed")
	}
	return client
}



func OpenCollection(collectionName string) *mongo.Collection {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning unable to find .env file! ")
	}
	databaseName := os.Getenv("DATABASE_NAME")

	fmt.Println("DATABASE_NAME :", databaseName)

	collection := Client.Database(databaseName).Collection(collectionName)
	if collection == nil {
		return nil
	}
	return collection
}
