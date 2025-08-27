package grpc

import (
	"context"
	"crypto_trading_system/api/proto/user"
	"crypto_trading_system/internal/user/model"
	"crypto_trading_system/internal/user/service"
	"crypto_trading_system/pkg/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserGRPCServer struct {
	user.UnimplementedUserServiceServer
	userService service.UserService
	logger      *logger.Logger
}

func NewUserGRPCServer(userService service.UserService, logger *logger.Logger) *UserGRPCServer {
	return &UserGRPCServer{
		userService: userService,
		logger:      logger,
	}
}

func (s *UserGRPCServer) Register(ctx context.Context, req *user.RegisterRequest) (*user.RegisterResponse, error) {
	s.logger.WithContext(ctx).WithField("username", req.Username).Info("gRPC: rRegister request receiverd")

	// 转换请求
	serviceReq := service.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	}

	serviceResp, err := s.userService.Register(ctx, &serviceReq)
	if err != nil {
		s.logger.WithError(err).Error("Register service failed")
		return &user.RegisterResponse{
			Success: false,
			Message: err.Error(),
		}, status.Errorf(codes.Internal, "register failed: %v", err)
	}

	return &user.RegisterResponse{
		Success: true,
		Message: "Register successful",
		User:    s.convertUserToProto(serviceResp.User),
		Token:   serviceResp.Token,
	}, nil
}

func (s *UserGRPCServer) convertUserToProto(u *model.User) *user.User {
	protoUser := &user.User{
		Id:        uint64(u.ID),
		Email:     u.Email,
		Username:  u.Username,
		Status:    string(u.Status),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}

	if u.Profile != nil {
		protoUser.Profile = &user.UserProfile{
			Id:        uint64(u.Profile.ID),
			UserId:    uint64(u.Profile.UserID),
			FirstName: u.Profile.FirstName,
			LastName:  u.Profile.LastName,
			Phone:     u.Profile.Phone,
			Country:   u.Profile.Country,
			City:      u.Profile.City,
		}
	}

	return protoUser
}
