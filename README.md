# GitHub Release & Commit 监控机器人

一个 Telegram 机器人，用于监控 GitHub 仓库的 Release 发布和 Commit 提交，并推送通知。

## 功能特性

- ✅ 监控 GitHub 仓库的新 Release 发布
- ✅ 监控 GitHub 仓库的新 Commit 提交
- ✅ 灵活选择监控类型（Release、Commit 或两者）
- ✅ 自定义监控分支（main、master 或自定义）
- ✅ 支持多种通知方式（私聊或频道/群聊）
- ✅ 支持多个仓库同时监控

## 使用方法

### 1. 启动机器人

发送 `/start` 命令开始配置。

### 2. 添加仓库

1. 点击「添加仓库」按钮
2. 发送仓库地址，格式：`owner/repository`
3. 选择监控类型：
   - `Release` - 只监控新版本发布
   - `Commit` - 只监控新提交
   - `Release+Commit` - 同时监控两者
4. 如果选择了监控 Commit，选择要监控的分支：
   - `main` / `master` - 快速选择常用分支
   - `自定义分支` - 输入任意分支名
5. 选择通知方式：
   - `私聊通知` - 直接发送给管理员
   - `频道/群聊通知` - 发送到指定频道（需要将机器人设为管理员）

### 3. 查看已添加的仓库

发送 `/list` 命令或点击「查看已添加仓库」按钮。

### 4. 取消操作

在配置过程中，可以随时点击「取消」按钮或发送 `/cancel` 命令。

## 环境变量

- `TELEGRAM_BOT_TOKEN` - Telegram 机器人 Token
- `ADMIN_ID` - 管理员的 Telegram 用户 ID

## 部署

```bash
docker-compose up -d
```

## 配置文件

配置文件位于 `/data/configs.json`：

```json
[
    {
        "repo": "owner/repository",
        "channel_id": 0,
        "channel_title": "私聊",
        "monitor_releases": true,
        "monitor_commits": true,
        "branch": "main",
        "last_release_id": null,
        "last_commit_sha": null
    }
]
```

- `channel_id`: 为 0 表示私聊通知，否则为频道/群聊 ID
