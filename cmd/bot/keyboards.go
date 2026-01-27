package main

import "fmt"

// startKeyboard 构建主菜单按钮
func startKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"添加仓库","callback_data":"%s"}],[{"text":"查看已添加仓库","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`,
		callbackAddRepo, callbackListRepos, callbackCancel)
}

// cancelKeyboard 构建取消按钮
func cancelKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"取消","callback_data":"%s"}]]}`, callbackCancel)
}

// monitorTypeKeyboard 构建监控类型选择按钮
func monitorTypeKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"Release","callback_data":"%s"},{"text":"Commit","callback_data":"%s"}],[{"text":"Release+Commit","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`,
		callbackMonitorRelease, callbackMonitorCommit, callbackMonitorBoth, callbackCancel)
}

// branchKeyboard 构建分支选择按钮
func branchKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"main","callback_data":"%s"},{"text":"master","callback_data":"%s"}],[{"text":"自定义分支","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`,
		callbackBranchMain, callbackBranchMaster, callbackBranchCustom, callbackCancel)
}

// channelKeyboard 构建通知方式选择按钮
func channelKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"私聊通知","callback_data":"%s"}],[{"text":"频道/群聊通知","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`,
		callbackChannelPrivate, callbackChannelCustom, callbackCancel)
}
