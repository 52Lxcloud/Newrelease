package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		log.Fatal("FATAL: TELEGRAM_BOT_TOKEN is not set in the environment.")
	}

	adminIDRaw := strings.TrimSpace(os.Getenv("ADMIN_ID"))
	adminID, err := strconv.ParseInt(adminIDRaw, 10, 64)
	if err != nil {
		log.Fatal("FATAL: ADMIN_ID is not a valid integer or is not set in the environment.")
	}

	tg := newTelegramClient(token)
	me, err := tg.getMe()
	if err != nil {
		log.Fatalf("Failed to fetch bot info: %v", err)
	}
	tg.botID = me.ID

	log.Printf("Bot starting... Authorized Admin User ID is %d", adminID)
	go scheduledChecker(tg, adminID)

	offset := 0
	for {
		updates, err := tg.getUpdates(offset)
		if err != nil {
			log.Printf("Failed to fetch updates: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, upd := range updates {
			if upd.UpdateID >= offset {
				offset = upd.UpdateID + 1
			}

			// 处理回调查询
			if upd.CallbackQuery != nil {
				cb := upd.CallbackQuery
				if cb.From == nil {
					continue
				}
				if cb.From.ID != adminID {
					log.Printf("Unauthorized access attempt by user %d (%s %s)", cb.From.ID, cb.From.FirstName, cb.From.LastName)
					_ = tg.answerCallbackQuery(cb.ID, "没有权限", false)
					continue
				}
				handleCallbackQuery(tg, cb, adminID)
				continue
			}

			// 处理文本消息
			if upd.Message == nil || upd.Message.From == nil || upd.Message.Chat == nil {
				continue
			}

			fromID := upd.Message.From.ID
			if fromID != adminID {
				log.Printf("Unauthorized access attempt by user %d (%s %s)", fromID, upd.Message.From.FirstName, upd.Message.From.LastName)
				continue
			}

			handleMessage(tg, upd.Message, adminID)
		}
	}
}
