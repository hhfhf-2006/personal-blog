// Package timeutil 提供全局统一的北京时间（UTC+8 / 东八区）处理。
//
// 设计目标：让文章、评论等所有对外暴露的时间戳，无论数据库或宿主机时区如何，
// 都被「强制」按北京时间读取与展示，彻底消除跨时区导致的 ±8 小时偏差。
package timeutil

import "time"

// Shanghai 是东八区（北京时间，UTC+8）时区。
//
// 优先加载 IANA 时区数据库中的 "Asia/Shanghai"；在缺少 tzdata 的精简容器镜像中
// 回退到固定 +08:00 偏移，保证此变量永不为 nil。
// 中国自 1991 年起不再使用夏令时，固定偏移与 Asia/Shanghai 在当下完全等价。
var Shanghai = mustLoadShanghai()

func mustLoadShanghai() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*60*60)
}

// Now 返回当前北京时间。
//
// 所有写入数据库的时间戳（CreatedAt / UpdatedAt 等）都应使用它，
// 确保数据库 `timestamp` 列中存储的「墙上时钟」始终是北京时间。
func Now() time.Time {
	return time.Now().In(Shanghai)
}

// ToBeijing 将任意 time.Time 规范化为北京时间，用于对外输出（JSON 序列化）。
//
// 文章、评论的 created_at / updated_at 在数据库中为 `timestamp without time zone`，
// 驱动读回时会以 UTC 作为占位时区返回其「墙上时钟」。本函数取墙上时钟的
// 年月日时分秒，强制重新标注为东八区，从而：
//   - 对刚由 Now() 写入、本身已带东八区位置的时间是幂等的（值不变）；
//   - 对从数据库读回、被标注为 UTC 的裸时间，正确地重新解释为北京时间。
//
// 输出的 JSON 始终携带 +08:00 偏移，前端据此稳定显示为北京时间。
func ToBeijing(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
		Shanghai,
	)
}
