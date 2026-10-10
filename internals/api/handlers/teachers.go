package handlers

import (
	"context"
	"fmt"
	"grpcapi/internals/models"
	"grpcapi/internals/repositories/mongodb"
	"grpcapi/pkg/utils"
	pb "grpcapi/proto/gen"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) AddTeachers(ctx context.Context, req *pb.Teachers) (*pb.Teachers, error) {

	for _, teacher := range req.GetTeachers() {
		if teacher.Id != "" {
			return nil, status.Error(codes.InvalidArgument, "request is in incorrect format: non-empty ID fields are not allowed.")
		}
	}

	addedTeachers, err := mongodb.AddTeachersToDb(ctx, req.GetTeachers())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Teachers{Teachers: addedTeachers}, nil
}

func (s *Server) GetTeachers(ctx context.Context, req *pb.GetTeachersRequest) (*pb.Teachers, error) {
	// Filtering -- getting the filters from the request
	filter, err := buildFilter(req.Teacher, &models.Teacher{})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Sorting -- getting the sort options from the request
	sortOptions := buildSortOptions(req.GetSortBy())

	// Access the database to fetch data
	teachers, err := mongodb.GetTeachersFromDb(ctx, sortOptions, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Teachers{Teachers: teachers}, nil
}

func (s *Server) UpdateTeachers(ctx context.Context, req *pb.Teachers) (*pb.Teachers, error) {
	updatedTeachers, err := mongodb.ModifyTeachersInDb(ctx, req.Teachers)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Teachers{Teachers: updatedTeachers}, nil
}

func (s *Server) DeleteTeachers(ctx context.Context, req *pb.TeacherIds) (*pb.DeleteTeachersConfirmation, error) {
	ids := req.GetIds()
	var teacherIdsToDelete []string
	for _, v := range ids {
		teacherIdsToDelete = append(teacherIdsToDelete, v.Id)
	}

	client, err := mongodb.CreateMongoClient()
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

	return &pb.DeleteTeachersConfirmation{
		Status:     "Teachers successfully deleted",
		DeletedIds: deletedIds,
	}, nil
}
