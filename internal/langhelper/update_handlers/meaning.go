package update_handlers

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
)

func (uh *UpdateHandler) HandleMeaning(ctx context.Context, text string, chatID int64) error {
	entry := logrus.WithFields(logrus.Fields{
		"spot": "UpdateHandler.HandleMeaning",
	})
	words := strings.Split(strings.TrimSpace(text), " ")
	if len(words) < 2 {
		return errors.New("invalid command")
	}

	word, err := uh.wordsRepo.GetByWords(ctx, strings.ToLower(words[1]))
	if err != nil {
		entry.WithError(err).Error("failed to get word")
		return err
	}

	if words[0] == MeaningWithExampleCommand {
		f := tgbotapi.FileID(word.FileID)
		if _, err = uh.updateFetcher.GetBot().Send(&tgbotapi.PhotoConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChatID: chatID,
				},
				File: f,
			},
			Caption: fmt.Sprintf("%s\n%s", cases.Title(language.English).String(word.Word), word.Meaning),
		}); err != nil {
			entry.WithError(err).Error("failed to send message")
			return err
		}

		return nil
	}

	if _, err = uh.updateFetcher.GetBot().Send(&tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: chatID,
		},
		Text: fmt.Sprintf("%s\n%s", cases.Title(language.English).String(word.Word), word.Meaning),
	}); err != nil {
		entry.WithError(err).Error("failed to send message")
		return err
	}

	return nil
}
