package dao

import (
	"errors"

	"personal-blog-backend/internal/dao/model"

	"gorm.io/gorm"
)

type LikeDAO struct {
	db *gorm.DB
}

func NewLikeDAO(db *gorm.DB) *LikeDAO {
	return &LikeDAO{db: db}
}

// DB 暴露底层 *gorm.DB，用于跨 DAO 的事务编排
func (d *LikeDAO) DB() *gorm.DB { return d.db }

// Toggle 切换点赞状态：如果已经点过赞就取消，没点过就点赞
// 返回 true 表示操作后是"已点赞"，false 表示"已取消"
//
// 策略：先尝试 INSERT，如果发生 UNIQUE 冲突说明已存在记录，则改为 DELETE。
// 相比 SELECT FOR UPDATE 方案，INSERT-then-DELETE 天然防竞态：
//   - 并发两个"点赞"请求 → 一个 INSERT 成功（liked=true），另一个冲突后 DELETE（liked=false）
//   - 总效果取决于 PostgreSQL 的 UNIQUE 约束保证的原子性
func (d *LikeDAO) Toggle(like *model.Like) (liked bool, err error) {
	// 先尝试插入（带唯一约束 unique_user_like）。
	// 注意：必须用具名返回值 err（赋值 =），不可在 if 内用 := 重新声明；
	// 否则下面的唯一键冲突判断读到的是外层始终为 nil 的 err，
	// 取消点赞（DELETE）分支将永远不会执行 —— 这正是「取消点赞总数不减」的根因。
	err = d.db.Create(like).Error
	if err == nil {
		return true, nil // 插入成功 → 已点赞
	}

	// 插入失败但不是唯一键冲突 → 真正的数据库错误，向上返回
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		return false, err
	}

	// 唯一键冲突 → 已有点赞记录，执行取消（删除）
	result := d.db.Where("user_id = ? AND target_type = ? AND target_id = ?",
		like.UserID, like.TargetType, like.TargetID).
		Delete(&model.Like{})
	if result.Error != nil {
		return false, result.Error
	}
	// 删除成功，或并发下记录已被他人删除（RowsAffected==0）→ 均视为「已取消点赞」
	return false, nil
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

// DeleteByTargets 批量删除指定类型、指定 ID 列表的所有点赞记录
// 用于文章/评论被删除时清理关联的点赞数据
func (d *LikeDAO) DeleteByTargets(targetType model.LikeTargetType, targetIDs []int64) error {
	if len(targetIDs) == 0 {
		return nil
	}
	return d.db.Where("target_type = ? AND target_id IN ?", targetType, targetIDs).Delete(&model.Like{}).Error
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
