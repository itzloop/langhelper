package backup_handler

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itzloop/langhelperbot/internal/tgapi"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type BackupHandler struct {
	interval      time.Duration
	dbPath        string
	chatID        int64
	updateFetcher *tgapi.UpdateFetcher
}

func NewBackupHandler(chatID int64, dbPath string, interval time.Duration, updateFetcher *tgapi.UpdateFetcher) *BackupHandler {
	return &BackupHandler{
		interval:      interval,
		dbPath:        dbPath,
		chatID:        chatID,
		updateFetcher: updateFetcher,
	}
}

func (bh *BackupHandler) Start(ctx context.Context) (err error) {
	entry := logrus.WithFields(logrus.Fields{
		"spot":     "BackupHandler.Start",
		"chat_id":  bh.chatID,
		"interval": bh.interval,
		"dbPath":   bh.dbPath,
	})

	entry.Info("running backup handler")
	if err = bh.updateFetcher.BlockTillStarted(ctx); err != nil {
		entry.WithError(err).Error("couldn't wait for UpdateFetcher to start")
		return err
	}

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("BackupHandler recovered: %v", e)
		}

		bh.backup(entry)
	}()

	ticker := time.NewTicker(bh.interval)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				return ctx.Err()
			}

			return nil
		case <-ticker.C:
			entry.Info("creating a backup")
			bh.backup(entry)
		}
	}
}

func (bh *BackupHandler) backup(entry *logrus.Entry) {
	f, err := os.ReadFile(bh.dbPath)
	if err != nil {
		entry.WithError(err).Error("failed to read db file")
	}

	name := fmt.Sprintf("sqlite_backup_%s.db", time.Now().In(time.UTC).Format(time.DateOnly))
	fb := tgbotapi.FileBytes{
		Name:  name,
		Bytes: f,
	}

	for i := 0; i < 5; i++ {
		if _, err = bh.updateFetcher.GetBot().Send(&tgbotapi.DocumentConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChatID: bh.chatID,
				},
				File: fb,
			},
		}); err != nil {
			entry.WithError(err).Error("failed to send backup to server. retrying...")
			continue
		}

		break
	}
}
