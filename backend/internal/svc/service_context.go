package svc

import (
	"personal-blog-backend/internal/config"
	"personal-blog-backend/internal/dao"
	commentcontroller "personal-blog-backend/internal/controller/comment"
	likecontroller "personal-blog-backend/internal/controller/like"
	postcontroller "personal-blog-backend/internal/controller/post"
	usercontroller "personal-blog-backend/internal/controller/user"
	"personal-blog-backend/internal/infra/db"
	commentservice "personal-blog-backend/internal/service/comment"
	likeservice "personal-blog-backend/internal/service/like"
	postservice "personal-blog-backend/internal/service/post"
	userservice "personal-blog-backend/internal/service/user"

	"gorm.io/gorm"
)

type ServiceContext struct {
	Config           config.Config
	DB               *gorm.DB
	UserController   *usercontroller.Controller
	PostController   *postcontroller.Controller
	CommentController *commentcontroller.Controller
	LikeController   *likecontroller.Controller
}

func NewServiceContext(cfg config.Config) (*ServiceContext, error) {
	database, err := db.NewPostgres(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	// —— DAO 层 ——
	userDAO := dao.NewUserDAO(database)
	postDAO := dao.NewPostDAO(database)
	commentDAO := dao.NewCommentDAO(database)
	likeDAO := dao.NewLikeDAO(database)

	// —— Service 层 ——
	userService := userservice.NewService(userDAO, cfg.JWTSecret)
	postService := postservice.NewService(postDAO, commentDAO, likeDAO, userDAO)
	commentService := commentservice.NewService(commentDAO, userDAO, likeDAO)
	likeService := likeservice.NewService(likeDAO)

	// —— Controller 层 ——
	userController := usercontroller.NewController(userService)
	postController := postcontroller.NewController(postService)
	commentController := commentcontroller.NewController(commentService)
	likeController := likecontroller.NewController(likeService)

	return &ServiceContext{
		Config:           cfg,
		DB:               database,
		UserController:   userController,
		PostController:   postController,
		CommentController: commentController,
		LikeController:   likeController,
	}, nil
}

