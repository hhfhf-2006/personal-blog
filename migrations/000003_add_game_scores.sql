-- ============================================================
-- 游戏分数模块 schema —— 合并自原 000003 ~ 000007
-- ============================================================
-- 设计要点：
--   1. 单记录 UPSERT 模型：每个用户每个游戏只保留一条最高分记录
--      （唯一约束 user_id + game_name），配合后端"仅当当前分 > 历史最高分才更新"。
--   2. achieved_at 与 created_at 分离：
--        created_at  —— 记录创建时间（注册时写入，恒定不变）
--        achieved_at —— 最佳成绩达成时间（每次刷新最高分时更新为服务器当前时间）
--   3. 用户注册时由触发器自动创建 score=0 的默认记录；排行榜查询过滤 score=0。
--
-- 本文件全程幂等（IF NOT EXISTS / CREATE OR REPLACE / ON CONFLICT），可安全重复执行。

-- —— 1. 游戏分数表（建表即包含 achieved_at，无需后续 ALTER）——
CREATE TABLE IF NOT EXISTS game_scores (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    game_name   VARCHAR(50) NOT NULL,
    score       BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),    -- 记录创建时间，不变
    achieved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- 最佳成绩达成时间

    CONSTRAINT fk_game_scores_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    -- 唯一约束：每个用户每个游戏一条记录。
    -- 其自带的唯一索引同时服务 "WHERE user_id=? AND game_name=?" 查询，
    -- 故不再单独建 idx_game_scores_user_game（避免冗余索引）。
    CONSTRAINT unique_user_game UNIQUE (user_id, game_name)
);

-- —— 2. 排行榜复合索引：完整覆盖
--    "WHERE game_name=? AND score>0 ORDER BY score DESC, achieved_at ASC"
-- 替代原 000003/000006/000007 中多次 DROP/CREATE 的索引演进。
CREATE INDEX IF NOT EXISTS idx_game_score
    ON game_scores (game_name, score DESC, achieved_at ASC);

-- —— 3. 触发器函数：用户注册时自动创建默认游戏记录（score=0）——
CREATE OR REPLACE FUNCTION create_default_game_scores()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO game_scores (user_id, game_name, score, created_at, achieved_at)
    VALUES (NEW.id, '2048', 0, NEW.created_at, NEW.created_at)
    ON CONFLICT (user_id, game_name) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- —— 4. 绑定触发器到 users 表（幂等：已存在则跳过）——
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgname = 'trg_create_default_game_scores'
          AND tgrelid = 'users'::regclass
    ) THEN
        CREATE TRIGGER trg_create_default_game_scores
            AFTER INSERT ON users
            FOR EACH ROW
            EXECUTE FUNCTION create_default_game_scores();
    END IF;
END $$;

-- —— 5. 为已存在用户补建默认记录（幂等）——
INSERT INTO game_scores (user_id, game_name, score, created_at, achieved_at)
SELECT u.id, '2048', 0, u.created_at, u.created_at
FROM users u
ON CONFLICT (user_id, game_name) DO NOTHING;
