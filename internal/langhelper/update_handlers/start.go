package update_handlers

import (
	"context"
	"github.com/itzloop/langhelperbot/internal/langhelper/db"
	"github.com/sirupsen/logrus"
	"time"
)

func (uh *UpdateHandler) HandleStart(ctx context.Context, userID int64) error {
	entry := logrus.WithFields(logrus.Fields{
		"spot":    "UpdateHandler.HandleStart",
		"user_id": userID,
	})

	if err := uh.usersRepo.Insert(ctx, db.UsersModel{
		UserID:    userID,
		CreatedAt: time.Now().In(time.UTC),
	}); err != nil {
		entry.WithError(err).Error("failed to insert new user")
		return err
	}

	// TODO get all the words
	words, err := uh.wordsRepo.GetAllWords(ctx)
	if err != nil {
		entry.WithError(err).Error("failed to get all words")
		return err
	}

	if len(words) == 0 {
		return nil
	}

	// TODO insert all the words in userWords
	if err = uh.userWordsRepo.InsertBulkSingleUser(ctx, userID, words); err != nil {
		entry.WithError(err).Error("failed to insert bulk single user")
		return err
	}

	return nil
}
