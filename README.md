# Newrelease

轻量级 Telegram 机器人，监控 GitHub 仓库的 Release 和 Commit。

## 部署

### 1. 运行方式

**方式 A: Docker Compose (推荐)**

直接在 `docker-compose.yml` 中配置环境变量：

```yaml
services:
  bot:
    image: 52lxcloud/newrelease:latest
    restart: unless-stopped
    environment:
      - TELEGRAM_BOT_TOKEN=your_token
      - ADMIN_ID=123456789
      - GITHUB_TOKEN=ghp_xxxx
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

- **频率限制**: 未配置 GitHub Token 时每小时 60 次请求，配置后 5000 次。
- **私有仓库**: 如果提供了带 `repo` 权限的 Token，支持监控私有仓库。
- **数据存储**: 数据保存在 `data/` 目录下，重启容器数据不丢失。

## 许可证

MIT License
