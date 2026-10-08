package handlers

import (
	"context"
	"fmt"
	"grpcapi/internals/models"
	"grpcapi/internals/repositories/mongodb"
	"grpcapi/pkg/utils"
	pb "grpcapi/proto/gen"
	"reflect"
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
	filter, err := buildFilterForTeachers(req.Teacher)
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

func buildFilterForTeachers(teacherObj *pb.Teacher) (bson.M, error) {
	filter := bson.M{}

	if teacherObj == nil {
		return filter, nil
	}

	var modelTeacher models.Teacher
	modelVal := reflect.ValueOf(&modelTeacher).Elem()
	modelType := modelVal.Type()

	reqVal := reflect.ValueOf(teacherObj).Elem()
	reqType := reqVal.Type()

	for i := 0; i < reqVal.NumField(); i++ {
		fieldVal := reqVal.Field(i)
		fieldName := reqType.Field(i).Name

		if fieldVal.IsValid() && !fieldVal.IsZero() {
			modelField := modelVal.FieldByName(fieldName)
			if modelField.IsValid() && modelField.CanSet() {
				modelField.Set(fieldVal)
			}
		}
	}

	// Iterate over the modelTeacher to build the filter using bson.M
	for i := 0; i < modelVal.NumField(); i++ {
		fieldVal := modelVal.Field(i)

		if fieldVal.IsValid() && !fieldVal.IsZero() {
			bsonTag := modelType.Field(i).Tag.Get("bson")
			bsonTag = strings.TrimSuffix(bsonTag, ",omitempty")
			if bsonTag == "_id" {
				objId, err := primitive.ObjectIDFromHex(teacherObj.Id)
				if err != nil {
					return nil, utils.ErrorHandler(err, "Invalid Id")
				}
				filter[bsonTag] = objId
			} else {
				filter[bsonTag] = fieldVal.Interface().(string)
			}
		}
	}

	fmt.Println("Filter:", filter)
	return filter, nil
}

func buildSortOptions(sortFields []*pb.SortField) bson.D {
	var sortOptions bson.D

	for _, sortField := range sortFields {
		order := 1 // default ascending
		if sortField.GetOrder() == pb.Order_DESC {
			order = -1
		}
		sortOptions = append(sortOptions, bson.E{Key: sortField.Field, Value: order})
	}

	fmt.Println("Sort Options:", sortOptions)
	return sortOptions
}
