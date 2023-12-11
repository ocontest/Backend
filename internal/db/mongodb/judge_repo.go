package mongodb

import (
	"context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"ocontest/internal/db"
	"ocontest/pkg"
	"ocontest/pkg/configs"
	"ocontest/pkg/structs"
)

// Replace the placeholder with your Atlas connection string
type JudgeRepoImp struct {
	collection *mongo.Collection
}

func NewJudgeRepo(config configs.SectionMongo) (db.JudgeRepo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(config.Address).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &JudgeRepoImp{
		collection: client.Database(config.Database).Collection(collectionName),
	}, client.Ping(ctx, nil)
}

func (j JudgeRepoImp) Insert(ctx context.Context, response structs.JudgeResponse) (string, error) {
	//TODO implement me

	document := bson.D{
		{"userError", response.UserError},
		{"testStates", response.TestStates},
	}

	res, err := j.collection.InsertOne(context.Background(), document)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (j JudgeRepoImp) GetResults(ctx context.Context, id string) (structs.JudgeResponse, error) {

	fid, err := primitive.ObjectIDFromHex(id)
	var ans structs.JudgeResponse
	if err != nil {
		return ans, err
	}

	err = j.collection.FindOne(context.Background(), bson.D{{"_id", fid}}, nil).Decode(&ans)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ans, pkg.ErrNotFound
		}
		return ans, err
	}
	return ans, nil
}