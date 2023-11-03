package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"ocontest/pkg/structs"
)

type AuthRepo interface {
	InsertUser(ctx context.Context, user structs.User) (int64, error)
}
type AuthRepoImp struct {
	conn *pgxpool.Pool
}

func NewAuthRepo(ctx context.Context, conn *pgxpool.Pool) (AuthRepo, error) {
	ans := &AuthRepoImp{conn: conn}
	return ans, ans.Migrate(ctx)
}

func (a *AuthRepoImp) Migrate(ctx context.Context) error {
	stmt := `
	CREATE TABLE IF NOT EXISTS users(
	    id SERIAL PRIMARY KEY ,
	    username VARCHAR(40),
	    password varchar(70),
	    email varchar(40),
	    created_at TIMESTAMP DEFAULT NOW(),
	    is_verified boolean DEFAULT false,
	    UNIQUE (username),
	    UNIQUE (email)
	)
	`

	_, err := a.conn.Exec(ctx, stmt)
	return err
}
func (a *AuthRepoImp) InsertUser(ctx context.Context, user structs.User) (int64, error) {
	stmt := `
	INSERT INTO users(username, password, email) VALUES($1, $2, $3) RETURNING id
	`
	var id int64
	err := a.conn.QueryRow(ctx, stmt, user.Username, user.EncryptedPassword, user.Email).Scan(&id)
	return id, err
}
