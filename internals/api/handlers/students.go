package handlers

import (
	"context"
	"grpcapi/internals/models"
	"grpcapi/internals/repositories/mongodb"
	pb "grpcapi/proto/gen"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) AddStudents(ctx context.Context, req *pb.Students) (*pb.Students, error) {

	for _, student := range req.GetStudents() {
		if student.Id != "" {
			return nil, status.Error(codes.InvalidArgument, "request is in incorrect format: non-empty ID fields are not allowed.")
		}
	}

	addedStudents, err := mongodb.AddStudentsToDb(ctx, req.GetStudents())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Students{Students: addedStudents}, nil
}

func (s *Server) GetStudents(ctx context.Context, req *pb.GetStudentsRequest) (*pb.Students, error) {
	// Filtering -- getting the filters from the request
	filter, err := buildFilter(req.Student, &models.Student{})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Sorting -- getting the sort options from the request
	sortOptions := buildSortOptions(req.GetSortBy())

	// Access the database to fetch data

	pageNumber := req.GetPageNumber()
	pageSize := req.GetPageSize()

	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	students, err := mongodb.GetStudentsFromDb(ctx, sortOptions, filter, pageNumber, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Students{Students: students}, nil
}

func (s *Server) UpdateStudents(ctx context.Context, req *pb.Students) (*pb.Students, error) {
	updatedStudents, err := mongodb.ModifyStudentsInDb(ctx, req.Students)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Students{Students: updatedStudents}, nil
}

func (s *Server) DeleteStudents(ctx context.Context, req *pb.StudentIds) (*pb.DeleteStudentsConfirmation, error) {
	ids := req.GetIds()
	var studentIdsToDelete []string
	for _, student := range ids {
		studentIdsToDelete = append(studentIdsToDelete, student)
	}

	deletedIds, err := mongodb.DeleteStudentsFromDb(ctx, studentIdsToDelete)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.DeleteStudentsConfirmation{
		Status:     "Students successfully deleted",
		DeletedIds: deletedIds,
	}, nil
}
