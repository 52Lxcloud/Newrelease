# Newrelease

轻量级 Telegram 机器人，监控 GitHub 仓库的 Release 和 Commit，支持 AI 自动翻译提交信息。

## 功能特性

- **Release 监控**：实时推送新版本发布通知。
- **Commit 监控**：监控指定分支的最新提交。
- **AI 翻译**：自动调用 AI 翻译英文 Commit Message
- **多渠道通知**：支持私聊或推送到频道/群组。
- **权限控制**：仅允许指定管理员操作。

## 部署

### 1. 运行方式

**方式 A: Docker Compose (推荐)**

创建 `docker-compose.yml`：

```yaml
services:
  bot:
    image: 52lxcloud/newrelease:latest
    restart: unless-stopped
    environment:
      # 基础配置
      - TELEGRAM_BOT_TOKEN=your_token
      - ADMIN_ID=123456789
      - GITHUB_TOKEN=ghp_xxxx
      
      # AI 翻译配置 (可选)
      # 如果不配置 API Key，将关闭翻译功能
      - AI_API_KEY=sk-xxxx
      - AI_BASE_URL=https://api.openai.com/v1  # 支持 DeepSeek, OpenAI 等兼容接口
      - AI_MODEL=gpt-5.2                 # 模型名称，如 deepseek-chat
    volumes:
      - ./data:/data
```

启动：
```bash
docker-compose up -d
```

**方式 B: 直接运行 Go 程序**

需要先创建一个 `.env` 文件配置环境变量：

```bash
TELEGRAM_BOT_TOKEN=your_token
ADMIN_ID=123456789
GITHUB_TOKEN=ghp_xxxx

# AI 配置 (可选)
AI_API_KEY=sk-xxxx
AI_BASE_URL=https://api.deepseek.com/v1
AI_MODEL=deepseek-chat
```

然后运行：
```bash
go run cmd/bot/*.go
```

## 使用命令

机器人仅响应 `ADMIN_ID` 配置的用户。

- `/add <owner/repo> [flags]` - 添加监控
- `/list` - 查看当前监控列表
- `/delete <id>` - 删除监控 (ID 来自 `/list`)
- `/help` - 显示帮助信息

### 添加监控示例

**基础用法** (监控 Release 和 Commit，获取默认分支):
```bash
/add kubernetes/kubernetes
```

**指定分支**:
```bash
/add kubernetes/kubernetes:release-1.28
```

**仅监控 Release**:
```bash
/add kubernetes/kubernetes -r
```

**仅监控 Commit**:
```bash
/add kubernetes/kubernetes -c
```

**通知到频道**:
1. 将机器人拉入频道并设为管理员。
2. 在命令最后加上频道 ID 或 username:
```bash
/add kubernetes/kubernetes @k8s_updates
```

## 说明

- **AI 翻译**: 
  - 自动识别是否为中文，若是则跳过翻译。
  - 保留 `feat`, `fix` 等前缀不翻译，保持专业性。
  - 支持所有兼容 OpenAI 格式的 API 接口（如 DeepSeek, Moonshot 等）。
- **频率限制**: 未配置 GitHub Token 时每小时 60 次请求，配置后 5000 次。
- **私有仓库**: 如果提供了带 `repo` 权限的 Token，支持监控私有仓库。
- **数据存储**: 数据保存在 `data/` 目录下，重启容器数据不丢失。

## 许可证

MIT License
