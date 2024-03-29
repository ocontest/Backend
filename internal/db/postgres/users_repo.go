package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ocontest/backend/internal/db/repos"
	"github.com/ocontest/backend/pkg/structs"
)

type UsersRepoImp struct {
	conn *pgxpool.Pool
}

func NewAuthRepo(ctx context.Context, conn *pgxpool.Pool) (repos.UsersRepo, error) {
	ans := &UsersRepoImp{conn: conn}
	return ans, ans.Migrate(ctx)
}

func (a *UsersRepoImp) Migrate(ctx context.Context) error {
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
func (a *UsersRepoImp) InsertUser(ctx context.Context, user structs.User) (int64, error) {
	stmt := `
	INSERT INTO users(username, password, email) VALUES($1, $2, $3) RETURNING id 
	`
	var id int64
	err := a.conn.QueryRow(ctx, stmt, user.Username, user.EncryptedPassword, user.Email).Scan(&id)
	return id, err
}

func (a *UsersRepoImp) VerifyUser(ctx context.Context, userID int64) error {
	stmt := `
	UPDATE users SET is_verified = true WHERE id = $1
	`
	_, err := a.conn.Exec(ctx, stmt, userID)
	return err
}

func (a *UsersRepoImp) GetByUsername(ctx context.Context, username string) (structs.User, error) {
	stmt := `
	SELECT id, username, password, email, is_verified FROM users WHERE username = $1 
	`
	var user structs.User
	err := a.conn.QueryRow(ctx, stmt, username).Scan(&user.ID, &user.Username, &user.EncryptedPassword, &user.Email, &user.Verified)
	return user, err
}

func (a *UsersRepoImp) GetByID(ctx context.Context, userID int64) (structs.User, error) {
	stmt := `
	SELECT id, username, password, email, is_verified FROM users WHERE id = $1 
	`
	var user structs.User
	err := a.conn.QueryRow(ctx, stmt, userID).Scan(&user.ID, &user.Username, &user.EncryptedPassword, &user.Email, &user.Verified)
	return user, err
}

func (a *UsersRepoImp) GetUsername(ctx context.Context, userID int64) (string, error) {
	stmt := `
	SELECT username FROM users WHERE id = $1 
	`
	var username string
	err := a.conn.QueryRow(ctx, stmt, userID).Scan(&username)
	return username, err
}

func (a *UsersRepoImp) GetByEmail(ctx context.Context, email string) (structs.User, error) {
	stmt := `
	SELECT id, username, password, email, is_verified FROM users WHERE email = $1 
	`
	var user structs.User
	err := a.conn.QueryRow(ctx, stmt, email).Scan(&user.ID, &user.Username, &user.EncryptedPassword, &user.Email, &user.Verified)
	return user, err
}

// TODO: find a suitable query builder to do this shit. sorry for this shitty code you are gonna see, I had no other idea.
// if you change this also change UpdateProblem since i just copied this :).
func (a *UsersRepoImp) UpdateUser(ctx context.Context, user structs.User) error {
	args := make([]interface{}, 0)
	args = append(args, user.ID)

	stmt := `
	UPDATE users SET
	`

	if user.Username != "" {
		args = append(args, user.Username)
		stmt += fmt.Sprintf("username = $%d", len(args))
	}
	if user.Email != "" {
		args = append(args, user.Email)
		if len(args) > 1 {
			stmt += ","
		}
		stmt += fmt.Sprintf("email = $%d", len(args))
	}
	if user.EncryptedPassword != "" {
		args = append(args, user.EncryptedPassword)
		stmt += fmt.Sprintf("password = $%d", len(args))
	}
	stmt += " WHERE id = $1"
	if len(args) == 0 {
		return nil
	}

	_, err := a.conn.Exec(ctx, stmt, args...)
	return err
}
