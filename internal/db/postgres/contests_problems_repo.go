package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ocontest/backend/internal/db"
)

type ContestsProblemsMetadataRepoImp struct {
	conn *pgxpool.Pool
}

func (c *ContestsProblemsMetadataRepoImp) Migrate(ctx context.Context) error {
	stmt := `
	CREATE TABLE IF NOT EXISTS contest_problems (
		contest_id int NOT NULL,
		problem_id int NOT NULL,
		PRIMARY KEY (contest_id, problem_id),
		FOREIGN KEY(contest_id) REFERENCES contests(id) ON DELETE CASCADE,
		FOREIGN KEY(problem_id) REFERENCES problems(id) ON DELETE CASCADE
	)
	`

	_, err := c.conn.Exec(ctx, stmt)
	return err
}

func NewContestsProblemsMetadataRepo(ctx context.Context, conn *pgxpool.Pool) (db.ContestsProblemsRepo, error) {
	ans := &ContestsProblemsMetadataRepoImp{conn: conn}
	return ans, ans.Migrate(ctx)
}

func (c *ContestsProblemsMetadataRepoImp) AddProblemToContest(ctx context.Context, contestID, problemID int64) error {
	insertContestProblemsStmt := `
		INSERT INTO contest_problems(
			contest_id, problem_id) 
		VALUES($1, $2)
	`

	_, err := c.conn.Exec(ctx, insertContestProblemsStmt, contestID, problemID)
	if err != nil {
		return err
	}

	return nil
}

func (c *ContestsProblemsMetadataRepoImp) GetContestProblems(ctx context.Context, id int64) ([]int64, error) {
	selectContestProblemsStmt := `
		SELECT problem_id FROM contest_problems WHERE contest_id = $1
	`

	rows, err := c.conn.Query(ctx, selectContestProblemsStmt, id)
	if err != nil {
		return make([]int64, 0), err
	}
	defer rows.Close()

	result := make([]int64, 0)

	for rows.Next() {
		var problemID int64
		err := rows.Scan(&problemID)
		if err != nil {
			return make([]int64, 0), err
		}

		result = append(result, problemID)
	}

	return result, nil
}
