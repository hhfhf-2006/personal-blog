package svc

import (
	"personal-blog-backend/internal/config"
	"personal-blog-backend/internal/dao"
	usercontroller "personal-blog-backend/internal/controller/user"
	"personal-blog-backend/internal/infra/db"
	userservice "personal-blog-backend/internal/service/user"

	"gorm.io/gorm"
)

type ServiceContext struct {
	Config         config.Config
	DB             *gorm.DB
	UserController *usercontroller.Controller
}

func NewServiceContext(cfg config.Config) (*ServiceContext, error) {
	database, err := db.NewPostgres(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	userDAO := dao.NewUserDAO(database)
	userService := userservice.NewService(userDAO)
	userController := usercontroller.NewController(userService)

	return &ServiceContext{
		Config:         cfg,
		DB:             database,
		UserController: userController,
	}, nil
}