package handlers

import (
	"context"
	"grpcapi/internals/models"
	"grpcapi/internals/repositories/mongodb"
	pb "grpcapi/proto/gen"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) AddExecs(ctx context.Context, req *pb.Execs) (*pb.Execs, error) {

	for _, exec := range req.GetExecs() {
		if exec.Id != "" {
			return nil, status.Error(codes.InvalidArgument, "request is in incorrect format: non-empty ID fields are not allowed.")
		}
	}

	addedExecs, err := mongodb.AddExecsToDb(ctx, req.GetExecs())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Execs{Execs: addedExecs}, nil
}

func (s *Server) GetExecs(ctx context.Context, req *pb.GetExecsRequest) (*pb.Execs, error) {
	// Filtering -- getting the filters from the request
	filter, err := buildFilter(req.Exec, &models.Exec{})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Sorting -- getting the sort options from the request
	sortOptions := buildSortOptions(req.GetSortBy())

	// Access the database to fetch data
	execs, err := mongodb.GetExecsFromDb(ctx, sortOptions, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Execs{Execs: execs}, nil
}

func (s *Server) UpdateExecs(ctx context.Context, req *pb.Execs) (*pb.Execs, error) {
	updatedExecs, err := mongodb.ModifyExecsInDb(ctx, req.Execs)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Execs{Execs: updatedExecs}, nil
}

func (s *Server) DeleteExecs(ctx context.Context, req *pb.ExecIds) (*pb.DeleteExecsConfirmation, error) {
	ids := req.GetIds()
	var execIdsToDelete []string
	for _, exec := range ids {
		execIdsToDelete = append(execIdsToDelete, exec)
	}

	deletedIds, err := mongodb.DeleteExecsFromDb(ctx, execIdsToDelete)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.DeleteExecsConfirmation{
		Status:     "Execs successfully deleted",
		DeletedIds: deletedIds,
	}, nil
}
