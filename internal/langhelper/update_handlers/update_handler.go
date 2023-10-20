package update_handlers

import (
	"context"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itzloop/langhelperbot/internal/langhelper/db"
	"github.com/itzloop/langhelperbot/internal/tgapi"
	"github.com/sirupsen/logrus"
	"strings"
)

const (
	StartCommand              string = "/start"
	TestCommand               string = "/test"
	RandomCommand             string = "/random"
	MeaningCommand            string = "/meaning"
	MeaningWithExampleCommand string = "/meaning_with_example"
)

var (
	commands = map[string]string{
		StartCommand:              "Starts the bot",
		RandomCommand:             "Gives you random word to answer",
		MeaningCommand:            "find meaning of a word /meaning <word>",
		MeaningWithExampleCommand: "gives an example for a word /meaning_with_example <word>",
	}
)

type UpdateHandler struct {
	updateFetcher *tgapi.UpdateFetcher
	wordsRepo     *db.WordsRepo
	userWordsRepo *db.UserWordsRepo
	usersRepo     *db.UsersRepo
}

func NewUpdateHandler(uf *tgapi.UpdateFetcher, wordsRepo *db.WordsRepo, userWordsRepo *db.UserWordsRepo, usersRepo *db.UsersRepo) *UpdateHandler {
	return &UpdateHandler{updateFetcher: uf, wordsRepo: wordsRepo, userWordsRepo: userWordsRepo, usersRepo: usersRepo}
}

func (uh *UpdateHandler) HandlerLoop(ctx context.Context) (err error) {
	entry := logrus.WithFields(logrus.Fields{
		"spot": "UpdateHandler.HandlerLoop",
	})

	if err := uh.updateFetcher.BlockTillStarted(ctx); err != nil {
		entry.WithError(err).Error("couldn't wait for UpdateFetcher to start")
		return err
	}

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("UpdateHandler recovered: %v", e)
		}
	}()

	if err := uh.updateFetcher.SetAvailableCommands(commands); err != nil {
		logrus.WithError(err).Error("failed to set available commands")
		return err
	}

	updateChannel := uh.updateFetcher.GetUpdateChan()
	var (
		msg *tgbotapi.Message
	)
	for update := range updateChannel {
		if update.Message != nil {
			msg = update.Message
		} else if update.ChannelPost != nil {
			msg = update.ChannelPost
		} else if update.CallbackQuery != nil {
			msg = update.CallbackQuery.Message
			msg.Text = update.CallbackQuery.Data
		} else {
			v, _ := json.Marshal(update)
			entry.WithField("update", string(v)).Warnln("unhandled update type")
			continue
		}

		switch msg.Text {
		case StartCommand:
			_ = uh.HandleStart(ctx, msg.Chat.ID)
		case RandomCommand:
			_ = uh.HandleRandom(ctx, msg.Chat.ID)
		//case TestCommand:
		//	panic("this is a test")
		default:
			if strings.Contains(msg.Text, MeaningCommand) {
				if err := uh.HandleMeaning(ctx, msg.Text, msg.Chat.ID); err != nil {
					entry.WithError(err).Error("failed to handle meaning command")
				}
				continue
			}

			// TODO handle words without a meaning
			if len(msg.Photo) == 0 {
				continue
			}

			if err := uh.HandleInsert(ctx, msg.Caption, msg.Photo[len(msg.Photo)-1].FileID); err != nil {
				entry.WithError(err).Error("failed to insert a new word")
			}
		}
	}

	return nil
}
