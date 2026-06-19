-- 让 password_hash 可以为空（GitHub OAuth 用户没有密码）
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- 添加 GitHub OAuth 相关字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_id BIGINT;

-- 添加唯一约束（PostgreSQL 不支持 ADD CONSTRAINT IF NOT EXISTS，用 DO 块模拟）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'unique_github_id'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users ADD CONSTRAINT unique_github_id UNIQUE (github_id);
    END IF;
END $$;
