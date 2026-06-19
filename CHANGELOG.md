# 更新日志 (CHANGELOG)

本项目所有重要变更记录于此文件。

## [Unreleased] - 2026-06-19

### 新增：CI/CD 模块（运维进阶）

引入基于 **GitHub Actions** 的完整持续集成 / 持续交付流水线，对应作业「运维」模块要求。

#### 持续集成 CI（`.github/workflows/ci.yml`）
触发时机：推送到 `main` 分支或对 `main` 提交 Pull Request。

- **后端检查任务**
  - 按 `backend/go.mod` 声明版本安装 Go 环境，并缓存依赖
  - `gofmt` 代码格式检查（仅警告提示，不阻断流水线）
  - `go vet` 静态分析
  - `go build ./...` 编译验证
  - `go test ./...` 单元测试
- **Docker 构建验证任务**
  - 使用根目录 `Dockerfile` 构建镜像（仅构建、不推送），确保容器化部署始终可用
  - 启用 GitHub Actions 构建缓存，加速后续构建

#### 持续交付 CD（`.github/workflows/cd.yml`）
触发时机：代码合入 `main` 分支，或推送 `v*` 版本标签（如 `v1.0.0`）。

- 自动登录 GitHub 容器仓库 **GHCR**（使用内置 `GITHUB_TOKEN`，无需额外配置密钥）
- 自动构建后端 Docker 镜像并推送到 `ghcr.io/hhfhf-2006/personal-blog`
- 自动生成多种镜像标签：分支名、语义化版本号、commit SHA，默认分支额外打 `latest`

### 说明
- 当前后端尚无单元测试，`go test` 步骤会通过（无测试视为成功）；后续补充测试后该步骤将自动生效。
- 部分文件未通过 `gofmt`（多为 Windows CRLF 行尾导致），建议本地执行 `gofmt -w ./backend` 后提交。
