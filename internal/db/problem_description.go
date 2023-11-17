package db

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"ocontest/pkg"
	"ocontest/pkg/configs"
	"ocontest/pkg/structs"
	"time"
)

// Replace the placeholder with your Atlas connection string
const timeout = time.Second * 30

type ProblemDescriptionsRepo interface {
	Save(description string) (string, error)
	Get(id string) (string, error)
}

type ProblemDescriptionRepoImp struct {
	collection *mongo.Collection
}

func NewProblemDescriptionRepo(config configs.SectionMongo) (ProblemDescriptionsRepo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(config.Address).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &ProblemDescriptionRepoImp{
		collection: client.Database(config.Database).Collection(config.Collection),
	}, nil
}

func (p ProblemDescriptionRepoImp) Save(description string) (string, error) {
	document := bson.D{
		{"description", description},
	}
	// insert into collection testc

	res, err := p.collection.InsertOne(context.Background(), document)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), nil

}

func (p ProblemDescriptionRepoImp) Get(id string) (string, error) {
	fid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return "", err
	}

	var result structs.ProblemDescription
	err = p.collection.FindOne(context.Background(), bson.D{{"_id", fid}}, nil).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", pkg.ErrNotFound
		}
		return "", err
	}
	return result.Description, nil
}