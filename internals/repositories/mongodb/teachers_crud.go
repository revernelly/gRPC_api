package mongodb

import (
	"context"
	"errors"
	"fmt"
	"grpcapi/internals/models"
	"grpcapi/pkg/utils"
	pb "grpcapi/proto/gen"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddTeachersToDb(ctx context.Context, teacherFromReq []*pb.Teacher) ([]*pb.Teacher, error) {
	client, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal error")
	}
	defer client.Disconnect(ctx)

	newTeachers := make([]*models.Teacher, len(teacherFromReq))
	for i, pbTeacher := range teacherFromReq {
		newTeachers[i] = mapPbTeacherToModelTeacher(pbTeacher)
	}

	var addedTeachers []*pb.Teacher
	for _, teacher := range newTeachers {
		result, err := client.Database("school").Collection("teachers").InsertOne(ctx, teacher)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding value to database")
		}

		objectId, ok := result.InsertedID.(primitive.ObjectID)
		if ok {
			teacher.Id = objectId.Hex()
		}

		pbTeacher := mapModelTeacherToPb(*teacher)
		addedTeachers = append(addedTeachers, pbTeacher)
	}
	return addedTeachers, nil
}

func GetTeachersFromDb(ctx context.Context, sortOptions bson.D, filter bson.M) ([]*pb.Teacher, error) {
	client, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Error")
	}
	defer client.Disconnect(ctx)

	coll := client.Database("school").Collection("teachers")
	var cursor *mongo.Cursor
	if len(sortOptions) < 1 {
		cursor, err = coll.Find(ctx, filter)
	} else {
		cursor, err = coll.Find(ctx, filter, options.Find().SetSort(sortOptions))
	}
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Error")
	}
	defer cursor.Close(ctx)

	teachers, err := decodeEntities(ctx, cursor, func() *pb.Teacher { return &pb.Teacher{} }, func() *models.Teacher {
		return &models.Teacher{}
	})
	if err != nil {
		return nil, err
	}
	return teachers, nil
}

func ModifyTeachersInDb(ctx context.Context, pbTeachers []*pb.Teacher) ([]*pb.Teacher, error) {
	client, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal error")
	}
	defer client.Disconnect(ctx)

	var updatedTeachers []*pb.Teacher

	for _, teacher := range pbTeachers {
		if teacher.Id == "" {
			return nil, utils.ErrorHandler(errors.New("id cannot be blank"), "ID cannot be blank")
		}
		modelTeacher := mapPbTeacherToModelTeacher(teacher)

		objId, err := primitive.ObjectIDFromHex(teacher.Id)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Invalid Id")
		}

		// Convert model Teacher to BSON document
		modelDoc, err := bson.Marshal(modelTeacher)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal error")
		}

		var updateDoc bson.M
		err = bson.Unmarshal(modelDoc, &updateDoc)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal error")
		}

		// Remove the "_id" field from the update document
		delete(updateDoc, "_id")

		_, err = client.Database("school").Collection("teachers").UpdateOne(ctx, bson.M{"_id": objId}, bson.M{"$set": updateDoc})
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprintln("error updating teacher id", teacher.Id))
		}

		updatedTeacher := mapModelTeacherToPb(*modelTeacher)

		updatedTeachers = append(updatedTeachers, updatedTeacher)
	}
	return updatedTeachers, nil
}

func DeleteTeachersFromDb(ctx context.Context, teacherIdsToDelete []string) ([]string, error) {
	client, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal error")
	}
	defer client.Disconnect(ctx)

	objectIds := make([]primitive.ObjectID, len(teacherIdsToDelete))
	for i, id := range teacherIdsToDelete {
		// Check if the ID is blank
		if strings.TrimSpace(id) == "" {
			return nil, utils.ErrorHandler(
				fmt.Errorf("teacher ID cannot be blank"),
				"teacher ID cannot be blank",
			)
		}

		// Convert Id to a mongoDB objectID
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprintf("incorrect id: %v", id))
		}
		objectIds[i] = objectId
	}

	filter := bson.M{"_id": bson.M{"$in": objectIds}}
	result, err := client.Database("school").Collection("teachers").DeleteMany(ctx, filter)
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Error")
	}

	if result.DeletedCount == 0 {
		return nil, utils.ErrorHandler(err, "No teachers were deleted. ids/Entries do not exist.")
	}

	deletedIds := make([]string, result.DeletedCount)
	for i, id := range objectIds {
		deletedIds[i] = id.Hex()
	}
	return deletedIds, nil
}
