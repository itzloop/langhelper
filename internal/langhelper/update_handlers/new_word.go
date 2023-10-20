package update_handlers

import (
	"context"
	"github.com/itzloop/langhelperbot/internal/langhelper/db"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func (uh *UpdateHandler) HandleInsert(ctx context.Context, caption, fileID string) error {
	entry := logrus.WithFields(logrus.Fields{
		"spot": "UpdateHandler.HandleInsert",
	})

	var (
		words   = strings.Split(caption, "\n")
		word    = strings.ToLower(words[0])
		meaning = words[1]
	)
	if err := uh.wordsRepo.Insert(ctx, db.WordsModel{
		Word:      word,
		Meaning:   meaning,
		FileID:    fileID,
		CreatedAt: time.Now().In(time.UTC),
	}); err != nil {
		entry.WithError(err).Error("failed to insert word to db")
		return err
	}

	users, err := uh.usersRepo.ListIDs(ctx)
	if err != nil {
		entry.WithError(err).Error("failed to list user ids")
		return err
	}

	if len(users) == 0 {
		return nil
	}

	if err = uh.userWordsRepo.InsertBulkSingleWord(ctx, word, users); err != nil {
		entry.WithError(err).Error("failed to bulk insert in user_words repo")
		return err
	}

	return nil
}
