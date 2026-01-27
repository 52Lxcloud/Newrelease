package main

import "sync"

// setupState 设置状态枚举
type setupState int

const (
	stateIdle setupState = iota
	stateWaitingRepo
	stateWaitingMonitorType
	stateWaitingBranch
	stateWaitingBranchCustom
	stateWaitingChannelType
	stateWaitingChannel
)

// setupSession 用户设置会话
type setupSession struct {
	state          setupState
	repo           string
	monitorRelease bool
	monitorCommit  bool
	branch         string
	lastBotMsgID   int
	chatID         int64
}

var (
	sessionMu sync.Mutex
	session   = setupSession{state: stateIdle}
)

// setSession 设置当前会话
func setSession(s setupSession) {
	sessionMu.Lock()
	session = s
	sessionMu.Unlock()
}

// getSession 获取当前会话
func getSession() setupSession {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	return session
}
