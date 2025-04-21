package postgres

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserPostgres struct {
	db *pgxpool.Pool
}

func NewUserPostgres(db *pgxpool.Pool) *UserPostgres {
	return &UserPostgres{db: db}
}

func (r *UserPostgres) UserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT u.id, u.username, u.password, u.email, array_agg(r.name)
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.id = $1
		GROUP BY u.id
	`

	row := r.db.QueryRow(ctx, query, id)
	var user models.User
	var roles []string

	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &roles)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, app_errors.ErrUserNotFound
		}
		return nil, err
	}

	user.Roles = roles
	return &user, nil
}

func (r *UserPostgres) UserByName(ctx context.Context, name string) (*models.User, error) {
	query := `
		SELECT u.id, u.username, u.password, u.email, array_agg(r.name)
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.username = $1
		GROUP BY u.id
	`

	row := r.db.QueryRow(ctx, query, name)
	var user models.User
	var roles []string

	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &roles)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, app_errors.ErrUserNotFound
		}
		return nil, err
	}

	user.Roles = roles
	return &user, nil
}

func (r *UserPostgres) CreateUser(ctx context.Context, user models.User) (*models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	queryUser := `INSERT INTO users (username, password, email) VALUES ($1, $2, $3) RETURNING id`
	var userID uuid.UUID
	err = tx.QueryRow(ctx, queryUser, user.Username, user.Password, user.Email).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if ok := (errors.As(err, &pgErr)); ok && pgErr.Code == "23505" {
			return nil, app_errors.ErrUserExists
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}
	user.ID = userID

	queryRole := `SELECT id FROM roles WHERE name = $1`
	insertUserRole := `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`
	for _, roleName := range user.Roles {
		var roleID int
		if err = tx.QueryRow(ctx, queryRole, roleName).Scan(&roleID); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, insertUserRole, userID, roleID); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &user, nil
}
