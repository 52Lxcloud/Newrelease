# Newrelease

轻量级 Telegram 机器人，监控 GitHub 仓库的 Release 和 Commit，支持 AI 自动翻译。

## 功能

- **Release 监控** - 新版本发布通知
- **Commit 监控** - 指定分支的提交通知
- **AI 翻译** - 自动翻译英文提交信息
- **话题支持** - 开启话题的群组自动按仓库创建话题
- **权限控制** - 仅管理员可操作

## 部署

### Docker Compose（推荐）

```yaml
services:
  bot:
    image: 52lxcloud/newrelease:latest
    restart: unless-stopped
    environment:
      - TELEGRAM_BOT_TOKEN=your_token
      - ADMIN_ID=123456789
      - GITHUB_TOKEN=ghp_xxxx          # 可选，提升 API 限额
      - AI_API_KEY=sk-xxxx             # 可选，不配置则关闭翻译
      - AI_BASE_URL=https://api.openai.com/v1
      - AI_MODEL=gpt-4o
    volumes:
      - ./data:/data
```

```bash
docker-compose up -d
```

### 直接运行

```bash
# 配置 .env 文件
TELEGRAM_BOT_TOKEN=your_token
ADMIN_ID=123456789

# 运行
go run cmd/bot/*.go
```

## 使用

机器人仅响应 `ADMIN_ID` 配置的用户。

| 命令 | 说明 |
|------|------|
| `/add <repo>` | 添加仓库监控 |
| `/list` | 查看监控列表 |
| `/delete <id>` | 删除监控 |
| `/help` | 显示帮助 |

### 示例

```bash
# 基础用法（监控 Release + Commit）
/add kubernetes/kubernetes

# 指定分支
/add kubernetes/kubernetes:release-1.28

# 仅监控 Release
/add kubernetes/kubernetes -r

# 仅监控 Commit
/add kubernetes/kubernetes -c

# 推送到群组（支持 @username 或群组 ID）
/add kubernetes/kubernetes @my_group
/add kubernetes/kubernetes -1001234567890
```

### 话题功能

如果群组开启了话题功能，机器人会自动以仓库名创建话题，每个仓库的更新推送到对应话题。

> 需要机器人拥有「管理话题」权限

## 配置

| 环境变量 | 必填 | 说明 |
|---------|------|------|
| `TELEGRAM_BOT_TOKEN` | ✅ | Bot Token |
| `ADMIN_ID` | ✅ | 管理员用户 ID |
| `GITHUB_TOKEN` | ❌ | 提升限额至 5000 次/小时 |
| `AI_API_KEY` | ❌ | AI 翻译 API Key |
| `AI_BASE_URL` | ❌ | AI API 地址（默认 OpenAI） |
| `AI_MODEL` | ❌ | 模型名称 |

## 说明

- **AI 翻译**：自动识别中文跳过，保留 `feat/fix` 等前缀，支持 OpenAI 兼容接口
- **GitHub 限额**：未配置 Token 60 次/小时，配置后 5000 次/小时
- **私有仓库**：需要带 `repo` 权限的 Token
- **数据存储**：`data/` 目录，重启不丢失

## License

MIT
