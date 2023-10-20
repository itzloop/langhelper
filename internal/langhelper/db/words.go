package db

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type (
	WordsModel struct {
		Word      string
		Meaning   string
		FileID    string
		CreatedAt time.Time
	}
	WordsRepo struct {
		db *sql.DB
	}
)

func NewWordsRepo(db *sql.DB) (*WordsRepo, error) {
	repo := &WordsRepo{db: db}
	err := repo.init(context.Background())
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (repo *WordsRepo) init(ctx context.Context) error {
	_, err := repo.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS words(
    word TEXT PRIMARY KEY,
    meaning TEXT,
    file_id TEXT,
	created_at TIMESTAMP
)`)

	return err
}

func (repo *WordsRepo) Insert(ctx context.Context, model WordsModel) error {
	_, err := repo.db.ExecContext(ctx, "INSERT INTO words (word, meaning, file_id, created_at) VALUES($1, $2, $3, $4)", model.Word, model.Meaning, model.FileID, model.CreatedAt)
	return err
}

func (repo *WordsRepo) GetAllWords(ctx context.Context) ([]WordsModel, error) {
	rows, err := repo.db.QueryContext(ctx, "SELECT * FROM words")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []WordsModel
	for rows.Next() {
		var res WordsModel
		if err = rows.Scan(&res.Word, &res.Meaning, &res.FileID, &res.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, res)
	}

	return list, nil
}

func (repo *WordsRepo) GetByWords(ctx context.Context, word string) (*WordsModel, error) {
	var res WordsModel
	if err := repo.db.QueryRowContext(ctx, "SELECT * FROM words WHERE word = $1", word).
		Scan(&res.Word, &res.Meaning, &res.FileID, &res.CreatedAt); err != nil {
		return nil, err
	}

	return &res, nil
}
