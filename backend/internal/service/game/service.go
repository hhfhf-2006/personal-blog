package game

import (
	"errors"
	"log"
	"time"

	"personal-blog-backend/internal/dao"
	"personal-blog-backend/internal/dao/model"
	"personal-blog-backend/internal/dto"
	"personal-blog-backend/internal/pkg/apperror"

	"gorm.io/gorm"
)

type Service struct {
	gameScoreDAO *dao.GameScoreDAO
}

func NewService(gameScoreDAO *dao.GameScoreDAO) *Service {
	return &Service{gameScoreDAO: gameScoreDAO}
}

// SubmitScore 提交游戏分数。
//
// 工程逻辑：
//   - 用户注册时由 DB 触发器自动创建 game_score（score=0, time=注册时间）
//   - 每个用户每个游戏只保留一条最高分记录
//   - 如果 newScore > 历史最高分 → 更新分数和达成时间
//   - 如果 newScore <= 历史最高分 → 不做任何操作（静默跳过）
//   - "尚无记录"分支保留作为防御性兜底（触发器失效等极端情况）
func (s *Service) SubmitScore(userID int64, gameName string, score int64) error {
	if score <= 0 {
		return apperror.BadRequest("分数必须大于0")
	}
	if score > 100_000_000 {
		return apperror.BadRequest("分数超出合理范围")
	}
	if gameName == "" {
		return apperror.BadRequest("游戏名称不能为空")
	}

	// 查找用户在此游戏的现有最高分记录
	existing, err := s.gameScoreDAO.FindByUserAndGame(userID, gameName)
	if err != nil {
		return apperror.WrapInternal(err)
	}

	if existing == nil {
		// 尚无记录 → 直接插入
		log.Printf("[GAME] Service: 用户 %d 在 %s 尚无记录，插入新记录 score=%d", userID, gameName, score)
		record := &model.GameScore{
			UserID:   userID,
			GameName: gameName,
			Score:    score,
		}
		if err := s.gameScoreDAO.Create(record); err != nil {
			log.Printf("[GAME] Service: 插入失败 userID=%d, err=%v", userID, err)
			// 并发场景：另一个请求同时为同一用户创建了该游戏的记录
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				existing, findErr := s.gameScoreDAO.FindByUserAndGame(userID, gameName)
				if findErr != nil {
					return apperror.WrapInternal(findErr)
				}
				if existing != nil && score > existing.Score {
					if updateErr := s.gameScoreDAO.UpdateScore(existing.ID, score, time.Now()); updateErr != nil {
						return apperror.WrapInternal(updateErr)
					}
					log.Printf("[GAME] Service: 并发冲突后更新成功 userID=%d, game=%s, score=%d", userID, gameName, score)
				}
				return nil
			}
			return apperror.WrapInternal(err)
		}
		log.Printf("[GAME] Service: 插入成功 userID=%d, game=%s, score=%d, newID=%d", userID, gameName, score, record.ID)
		return nil
	}

	// 已有记录，但新分数没有更高 → 静默跳过
	if score <= existing.Score {
		log.Printf("[GAME] Service: 用户 %d 的新分数 %d <= 历史最高 %d，跳过", userID, score, existing.Score)
		return nil
	}

	// 新分数更高 → 更新最高分和达成时间
	log.Printf("[GAME] Service: 用户 %d 的新分数 %d > 历史最高 %d，更新记录 id=%d", userID, score, existing.Score, existing.ID)
	if err := s.gameScoreDAO.UpdateScore(existing.ID, score, time.Now()); err != nil {
		log.Printf("[GAME] Service: 更新失败 id=%d, err=%v", existing.ID, err)
		return apperror.WrapInternal(err)
	}
	log.Printf("[GAME] Service: 更新成功 id=%d, newScore=%d", existing.ID, score)
	return nil
}

// GetLeaderboard 获取排行榜。
//
// 如果 userID > 0，还会额外返回该用户的个人最佳成绩和排名。
func (s *Service) GetLeaderboard(gameName string, limit int, userID int64) (*dto.LeaderboardResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if gameName == "" {
		gameName = "2048"
	}

	rows, err := s.gameScoreDAO.ListByGame(gameName, limit)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	totalPlayers, err := s.gameScoreDAO.CountPlayers(gameName)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	entries := make([]dto.LeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = dto.LeaderboardEntry{
			Rank:       int64(i + 1),
			Username:   row.Username,
			Score:      row.Score,
			AchievedAt: row.AchievedAt,
		}
	}

	resp := &dto.LeaderboardResponse{
		GameName:     gameName,
		Entries:      entries,
		TotalPlayers: totalPlayers,
	}

	// 用户已登录 → 补充个人最佳成绩
	if userID > 0 {
		bestScore, bestAchievedAt, err := s.gameScoreDAO.UserBestRecord(userID, gameName)
		if err != nil {
			return nil, apperror.WrapInternal(err)
		}
		if bestScore > 0 {
			rank, err := s.gameScoreDAO.UserRank(userID, gameName, bestScore, bestAchievedAt)
			if err != nil {
				return nil, apperror.WrapInternal(err)
			}
			resp.UserBest = &dto.UserBest{
				Score: bestScore,
				Rank:  rank,
			}
		}
	}

	return resp, nil
}

// GetMyScore 获取当前用户在指定游戏中的最高分及排名。
// 用于前端初始化 Best Score 显示和提交分数后确认。
func (s *Service) GetMyScore(userID int64, gameName string) (*dto.MyScoreResponse, error) {
	if gameName == "" {
		gameName = "2048"
	}

	bestScore, bestAchievedAt, err := s.gameScoreDAO.UserBestRecord(userID, gameName)
	if err != nil {
		return nil, apperror.WrapInternal(err)
	}

	var rank int64
	if bestScore > 0 {
		rank, err = s.gameScoreDAO.UserRank(userID, gameName, bestScore, bestAchievedAt)
		if err != nil {
			return nil, apperror.WrapInternal(err)
		}
	}

	return &dto.MyScoreResponse{
		GameName: gameName,
		Score:    bestScore,
		Rank:     rank,
	}, nil
}
