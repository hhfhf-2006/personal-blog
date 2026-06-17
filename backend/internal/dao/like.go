package dao

import (
	"personal-blog-backend/internal/dao/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LikeDAO struct {
	db *gorm.DB
}

func NewLikeDAO(db *gorm.DB) *LikeDAO {
	return &LikeDAO{db: db}
}

// Toggle 切换点赞状态：如果已经点过赞就取消，没点过就点赞
// 返回 true 表示操作后是"已点赞"，false 表示"已取消"
func (d *LikeDAO) Toggle(like *model.Like) (liked bool, err error) {
	// INSERT ... ON CONFLICT DO NOTHING，根据 RowsAffected 判断是新增还是取消
	result := d.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "target_type"}, {Name: "target_id"}},
		DoNothing: true,
	}).Create(like)

	if result.Error != nil {
		return false, result.Error
	}

	// RowsAffected > 0 说明是新增（之前没点过赞）
	if result.RowsAffected > 0 {
		return true, nil
	}

	// RowsAffected == 0 说明已存在 → 需要取消点赞
	err = d.db.Where("user_id = ? AND target_type = ? AND target_id = ?",
		like.UserID, like.TargetType, like.TargetID).Delete(&model.Like{}).Error
	return false, err
}

// CountByTarget 统计某个对象的点赞数
func (d *LikeDAO) CountByTarget(targetType model.LikeTargetType, targetID int64) (int64, error) {
	var count int64
	err := d.db.Model(&model.Like{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Count(&count).Error
	return count, err
}

// CountByTargets 批量统计多个对象的点赞数，返回 map[targetID]count
func (d *LikeDAO) CountByTargets(targetType model.LikeTargetType, targetIDs []int64) (map[int64]int64, error) {
	if len(targetIDs) == 0 {
		return map[int64]int64{}, nil
	}

	type row struct {
		TargetID int64
		Count    int64
	}

	var rows []row
	err := d.db.Model(&model.Like{}).
		Select("target_id, COUNT(*) AS count").
		Where("target_type = ? AND target_id IN ?", targetType, targetIDs).
		Group("target_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int64, len(rows))
	for _, r := range rows {
		result[r.TargetID] = r.Count
	}
	return result, nil
}

// Exists 检查用户是否已点赞某个对象
func (d *LikeDAO) Exists(userID int64, targetType model.LikeTargetType, targetID int64) (bool, error) {
	var count int64
	err := d.db.Model(&model.Like{}).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Count(&count).Error
	return count > 0, err
}

// LikedByUser 批量检查用户对多个对象的点赞状态，返回 set of liked targetIDs
func (d *LikeDAO) LikedByUser(userID int64, targetType model.LikeTargetType, targetIDs []int64) (map[int64]bool, error) {
	if len(targetIDs) == 0 {
		return map[int64]bool{}, nil
	}

	var likes []model.Like
	err := d.db.Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, targetType, targetIDs).
		Find(&likes).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]bool, len(likes))
	for _, l := range likes {
		result[l.TargetID] = true
	}
	return result, nil
}
