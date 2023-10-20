package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/itzloop/langhelperbot/internal/langhelper/backup_handler"
	"github.com/itzloop/langhelperbot/internal/langhelper/db"
	"github.com/itzloop/langhelperbot/internal/langhelper/update_handlers"
	"github.com/itzloop/langhelperbot/internal/tgapi"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
)

type Update struct {
	UpdateID int `json:"update_id,omitempty"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, gCtx := errgroup.WithContext(ctx)

	go signalHandler(cancel)
	var (
		botTokenEnv string
		botTokenArg *string
		logLevel    *string
		timeout     *time.Duration
		fetchLimit  *int
		envPath     *string
		dbPath      *string
	)

	botTokenArg = flag.String("token", "", "Token used to authenticate with telegram bot api [this takes precedence over BOT_TOKEN env]")
	logLevel = flag.String("log-level", "info", "Possible levels: panic, fatal, error, warn, info, debug, trace")
	timeout = flag.Duration("timeout", time.Second*5, "HTTP client timeout")
	fetchLimit = flag.Int("fetch-limit", 100, "Count of updates to fetch in each request")
	envPath = flag.String("env-path", "/data/.env", "Path to the .env file")
	dbPath = flag.String("db-path", "/data/sqlite.db", "Path to the sqlite3 db file")
	backupReceiver := flag.Int64("backup-receiver", 0, "Telegram userID to send backup to")
	backupInterval := flag.Duration("backup-interval", 24*time.Hour, "Interval to backup")
	backup := flag.Bool("backup", false, "Send sqlite db backup to an specified user in Telegram. Needs backup-receiver to be specified")
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Fatalln("failed to get working directory")
	}

	if strings.Contains(*dbPath, "$WD") {
		*dbPath = path.Join(wd, "sqlite.db")
	}

	if err := parseEnv(wd, envPath); err != nil {
		logrus.WithError(err).Error("failed to load env")
	}

	botTokenEnv = os.Getenv("BOT_TOKEN")
	if strings.TrimSpace(*botTokenArg) == "" {
		*botTokenArg = botTokenEnv
		if strings.TrimSpace(*botTokenArg) == "" {
			logrus.Fatalln("bot token must be specified with [BOT_TOKEN] env or passed as an argument [-token]")
		}
	}

	l, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to parse log level")
	}
	logrus.SetLevel(l)

	cfg := tgapi.UpdateFetcherConfig{
		Debug:    l >= logrus.DebugLevel,
		BotToken: *botTokenArg,
		Limit:    *fetchLimit,
		Timeout:  *timeout,
	}

	uf := tgapi.NewUpdateFetcher(cfg)

	// TODO make this an argument variable
	sqlDB, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to connect to db")
	}

	wordsRepo, err := db.NewWordsRepo(sqlDB)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to create WordsRepo")
	}

	userWordsRepo, err := db.NewUserWordRepo(sqlDB)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to create UserWordsRepo")
	}

	usersRepo, err := db.NewUsersRepo(sqlDB)
	if err != nil {
		logrus.WithError(err).Fatalln("failed to create UsersRepo")
	}
	uh := update_handlers.NewUpdateHandler(uf, wordsRepo, userWordsRepo, usersRepo)

	g.Go(func() error {
		return uf.Start(gCtx)
	})

	g.Go(func() error {
		return uh.HandlerLoop(gCtx)
	})

	if *backup {
		if *backupReceiver == 0 {
			logrus.Fatalln("backup-receiver must be set with -backup flag.")
		}

		bh := backup_handler.NewBackupHandler(*backupReceiver, *dbPath, *backupInterval, uf)
		g.Go(func() error {
			return bh.Start(gCtx)
		})
	}

	// wait for stuff
	if err := g.Wait(); err != nil {
		logrus.WithError(err).Errorln("one of the goroutines failed. waiting for 5 seconds")
		cancel()
		time.Sleep(time.Second * 5)
		os.Exit(1)
	}
}

func parseEnv(wd string, envPath *string) error {
	if strings.TrimSpace(*envPath) == "" || strings.Contains(*envPath, "$WD") {
		*envPath = path.Join(wd, ".env")
	}

	return godotenv.Load(*envPath)
}

func signalHandler(cancel context.CancelFunc) {
	var (
		signalChan = make(chan os.Signal, 1)
		sig        os.Signal
	)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig = <-signalChan
	logrus.WithField("signal", sig).Infoln("received interrupt")

	cancel()

	sig = <-signalChan
	logrus.WithField("signal", sig).Fatalln("force quiting")
}
