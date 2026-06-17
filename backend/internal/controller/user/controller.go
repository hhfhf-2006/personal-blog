package user

import userservice "personal-blog-backend/internal/service/user"

type Controller struct {
	userService *userservice.Service
}

func NewController(userService *userservice.Service) *Controller {
	return &Controller{
		userService: userService,
	}
}