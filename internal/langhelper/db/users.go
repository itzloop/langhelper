package db

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type (
	UsersModel struct {
		UserID    int64
		CreatedAt time.Time
	}
	UsersRepo struct {
		db *sql.DB
	}
)

func NewUsersRepo(db *sql.DB) (*UsersRepo, error) {
	repo := &UsersRepo{db: db}
	err := repo.init(context.Background())
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (repo *UsersRepo) init(ctx context.Context) error {
	_, err := repo.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS users(
    user_id BIGINT PRIMARY KEY,
    created_at TIMESTAMP
)`)

	return err
}

func (repo *UsersRepo) Insert(ctx context.Context, user UsersModel) error {
	_, err := repo.db.ExecContext(ctx, "INSERT INTO users (user_id, created_at) VALUES ($1, $2) ON CONFLICT DO NOTHING", user.UserID, user.CreatedAt)
	return err
}

func (repo *UsersRepo) ListIDs(ctx context.Context) ([]int64, error) {
	rows, err := repo.db.Query("SELECT user_id FROM users")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var (
		list []int64
	)
	for rows.Next() {
		var tmp int64
		if err = rows.Scan(&tmp); err != nil {
			return nil, err
		}

		list = append(list, tmp)
	}

	return list, nil
}
