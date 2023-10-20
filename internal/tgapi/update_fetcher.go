package tgapi

import (
	"context"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	defaultFetchLimit = 100
	defaultTimeout    = 15 * time.Second
)

type (
	UpdateFetcherConfig struct {
		Debug    bool
		BotToken string
		Limit    int
		Timeout  time.Duration
	}

	UpdateFetcher struct {
		config UpdateFetcherConfig

		updatesChan tgbotapi.UpdatesChannel
		bot         *tgbotapi.BotAPI
		isClosed    *atomic.Bool
		isStarted   *atomic.Bool
		pending     *atomic.Bool
	}
)

func NewUpdateFetcher(config UpdateFetcherConfig) *UpdateFetcher {
	if config.Limit <= 0 {
		config.Limit = defaultFetchLimit
	}

	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}

	return &UpdateFetcher{
		config:    config,
		isClosed:  &atomic.Bool{},
		isStarted: &atomic.Bool{},
		pending:   &atomic.Bool{},
	}
}

func (uf *UpdateFetcher) Start(ctx context.Context) (err error) {
	entry := logrus.WithFields(logrus.Fields{
		"spot": "UpdateFetcher.Start",
	})

	if uf.IsStarted() {
		entry.Warnln("already started")
		return nil
	}

	if uf.pending.Swap(true) {
		entry.Warnln("pending for previous start")
		return nil
	}
	defer func() { uf.pending.Store(false) }()

	bot, err := tgbotapi.NewBotAPIWithClient(uf.config.BotToken, tgbotapi.APIEndpoint, &http.Client{
		Timeout: uf.config.Timeout,
	})
	if err != nil {
		entry.WithError(err).Error("failed to create bot api")
		return err
	}

	uf.bot = bot
	bot.Debug = uf.config.Debug
	entry = entry.WithFields(logrus.Fields{
		"bot_username": bot.Self.UserName,
		"bot_id":       bot.Self.ID,
	})
	entry.Infoln("connected to telegram bot api")

	updateConfig := tgbotapi.UpdateConfig{
		Offset:         0,
		Limit:          uf.config.Limit,
		Timeout:        int(uf.config.Timeout.Seconds()),
		AllowedUpdates: nil,
	}

	uf.updatesChan = bot.GetUpdatesChan(updateConfig)
	uf.isStarted.Store(true)

	<-ctx.Done()
	if err = uf.Close(); err != nil {
		entry.WithError(err).Error("failed to close update fetcher")
		return err
	}

	return nil
}

func (uf *UpdateFetcher) GetUpdateChan() tgbotapi.UpdatesChannel {
	if !uf.IsStarted() {
		return nil
	}

	return uf.updatesChan
}

func (uf *UpdateFetcher) GetBot() *tgbotapi.BotAPI {
	if !uf.IsStarted() {
		return nil
	}

	return uf.bot
}

func (uf *UpdateFetcher) SetAvailableCommands(commandsMap map[string]string) error {
	entry := logrus.WithFields(logrus.Fields{
		"spot": "UpdateFetcher.SetAvailableCommands",
	})

	if !uf.IsStarted() {
		return errors.New("not started")
	}

	var commands []tgbotapi.BotCommand
	for command, description := range commandsMap {
		commands = append(commands, tgbotapi.BotCommand{
			Command:     command,
			Description: description,
		})
	}
	setMyCommandsConfig := tgbotapi.NewSetMyCommands(commands...)

	res, err := uf.bot.Request(setMyCommandsConfig)
	if err != nil {
		return err
	}

	if !res.Ok {
		entry.WithFields(logrus.Fields{
			"ok":         res.Ok,
			"error_code": res.ErrorCode,
		}).Error("failed to set available commands")
		return errors.New("failed to set available commands")
	}

	return nil
}

func (uf *UpdateFetcher) BlockTillStarted(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if uf.IsStarted() {
			break
		}
	}

	return nil
}

func (uf *UpdateFetcher) IsStarted() bool {
	return uf.isStarted.Load()
}

func (uf *UpdateFetcher) Close() error {
	entry := logrus.WithFields(logrus.Fields{
		"spot": "UpdateFetcher.Close",
	})

	if !uf.isStarted.Load() {
		entry.Warnln("not started yet")
		return nil
	}

	if uf.isClosed.Swap(true) {
		entry.Warnln("already closed")
		return nil
	}

	uf.bot.StopReceivingUpdates()

	return nil
}
