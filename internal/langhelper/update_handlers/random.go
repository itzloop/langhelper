package update_handlers

import (
	"context"
	"database/sql"

	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (uh *UpdateHandler) HandleRandom(ctx context.Context, userID int64) error {
	entry := logrus.WithFields(logrus.Fields{
		"spot":    "UpdateHandler.HandleRandom",
		"user_id": userID,
	})

	word, err := uh.userWordsRepo.GetRandomWord(ctx, userID)
	if err != nil && err != sql.ErrNoRows {
		entry.WithError(err).Errorln("failed to get a random word")
		return err
	} else if err == sql.ErrNoRows {
		if _, err = uh.updateFetcher.GetBot().Send(&tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{ChatID: userID},
			Text:     "You need to start the bot first to use this feature.",
		}); err != nil {
			entry.WithError(err).Error("failed to send message")
			return err
		}

		return nil
	}

	msg := tgbotapi.NewMessage(userID, cases.Title(language.English).String(word.Word))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Show Meaning", fmt.Sprintf("/meaning %s", word.Word)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Show Meaning (With Example)", fmt.Sprintf("/meaning_with_example %s", word.Word)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Next Word", "/random"),
		),
	)
	if _, err = uh.updateFetcher.GetBot().Send(msg); err != nil {
		entry.WithError(err).Error("failed to send random word")
		return err
	}

	return nil
}
