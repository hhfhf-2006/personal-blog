package svc

import (
	"strings"

	"personal-blog-backend/internal/config"
	"personal-blog-backend/internal/dao"
	commentcontroller "personal-blog-backend/internal/controller/comment"
	gamecontroller "personal-blog-backend/internal/controller/game"
	likecontroller "personal-blog-backend/internal/controller/like"
	postcontroller "personal-blog-backend/internal/controller/post"
	usercontroller "personal-blog-backend/internal/controller/user"
	"personal-blog-backend/internal/infra/db"
	"personal-blog-backend/internal/pkg/oauth"
	commentservice "personal-blog-backend/internal/service/comment"
	gameservice "personal-blog-backend/internal/service/game"
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
	GameController   *gamecontroller.Controller
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
	gameScoreDAO := dao.NewGameScoreDAO(database)

	// —— Service 层 ——
	userService := userservice.NewService(userDAO, cfg.JWTSecret, cfg.AdminEmail, cfg.AdminUsername, cfg.AdminPassword)
	postService := postservice.NewService(postDAO, commentDAO, likeDAO, userDAO)
	commentService := commentservice.NewService(commentDAO, userDAO, likeDAO, postDAO)
	likeService := likeservice.NewService(likeDAO, postDAO, commentDAO)
	gameService := gameservice.NewService(gameScoreDAO)

	// —— GitHub OAuth ——
	githubOAuth := oauth.NewClient(oauth.GitHubConfig{
		ClientID:     cfg.GitHub.ClientID,
		ClientSecret: cfg.GitHub.ClientSecret,
		RedirectURI:  cfg.GitHub.RedirectURI,
	})

	// —— Controller 层 ——
	// GitHub OAuth cookie Secure 标志：仅当回调 URL 使用 HTTPS 时启用（本地开发用 HTTP）
	secureCookie := strings.HasPrefix(cfg.GitHub.RedirectURI, "https://")
	userController := usercontroller.NewController(userService, githubOAuth, secureCookie)
	postController := postcontroller.NewController(postService)
	commentController := commentcontroller.NewController(commentService)
	likeController := likecontroller.NewController(likeService)
	gameController := gamecontroller.NewController(gameService)

	return &ServiceContext{
		Config:           cfg,
		DB:               database,
		UserController:   userController,
		PostController:   postController,
		CommentController: commentController,
		LikeController:   likeController,
		GameController:   gameController,
	}, nil
}

