package db

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strings"
	"time"
)

type (
	UserWordModel struct {
		UserID    int64
		Word      string
		LastAsked time.Time
	}

	UserWordsRepo struct {
		db *sql.DB
	}
)

func NewUserWordRepo(db *sql.DB) (*UserWordsRepo, error) {
	repo := &UserWordsRepo{db: db}
	err := repo.init(context.Background())
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (repo *UserWordsRepo) init(ctx context.Context) error {
	_, err := repo.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS user_words(
    user_id BIGINT REFERENCES users (user_id),
    word TEXT REFERENCES words (word),
    last_asked TIMESTAMP,
    PRIMARY KEY(user_id, word)
)`)

	return err
}

func (repo *UserWordsRepo) InsertBulkSingleUser(ctx context.Context, user int64, words []WordsModel) error {
	userWords := make([]UserWordModel, 0, cap(words))
	for _, word := range words {
		userWords = append(userWords, UserWordModel{
			UserID:    user,
			Word:      word.Word,
			LastAsked: time.Time{},
		})
	}

	return repo.InsertBulk(ctx, userWords)
}

func (repo *UserWordsRepo) InsertBulkSingleWord(ctx context.Context, word string, users []int64) error {
	userWords := make([]UserWordModel, 0, cap(users))
	for _, user := range users {
		userWords = append(userWords, UserWordModel{
			UserID:    user,
			Word:      word,
			LastAsked: time.Time{},
		})
	}

	return repo.InsertBulk(ctx, userWords)
}

func (repo *UserWordsRepo) InsertBulk(ctx context.Context, userWords []UserWordModel) error {
	valueStrings := make([]string, 0, len(userWords))
	valueArgs := make([]interface{}, 0, len(userWords)*3)
	for _, userWord := range userWords {
		valueStrings = append(valueStrings, "(?, ?, ?)")
		valueArgs = append(valueArgs, userWord.UserID)
		valueArgs = append(valueArgs, userWord.Word)
		valueArgs = append(valueArgs, userWord.LastAsked)
	}
	stmt := fmt.Sprintf("INSERT INTO user_words (user_id, word, last_asked) VALUES %s",
		strings.Join(valueStrings, ","))

	_, err := repo.db.ExecContext(ctx, stmt, valueArgs...)
	return err
}
func (repo *UserWordsRepo) GetRandomWord(ctx context.Context, userID int64) (*UserWordModel, error) {
	var userWord UserWordModel
	err := repo.db.QueryRowContext(ctx, `SELECT * FROM user_words WHERE user_id = $1 ORDER BY last_asked ASC LIMIT 1`, userID).
		Scan(&userWord.UserID, &userWord.Word, &userWord.LastAsked)
	if err != nil {
		return nil, err
	}

	// TODO this should be transaction or should be handled in a single query.
	// TODO I don't know if the latter is possible with sqlite.
	// TODO but this solution is good enough and i'm sticking to it :)
	_, err = repo.db.ExecContext(ctx, `UPDATE user_words SET last_asked = $1 WHERE user_id = $2 AND word = $3`, time.Now().In(time.UTC), userID, userWord.Word)
	if err != nil {
		return nil, err
	}

	return &userWord, nil
}
