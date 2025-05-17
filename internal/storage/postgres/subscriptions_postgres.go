package postgres

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionPostgres struct {
	db *pgxpool.Pool
}

func NewSubscriptionPostgres(db *pgxpool.Pool) *SubscriptionPostgres {
	return &SubscriptionPostgres{db: db}
}

func (r *SubscriptionPostgres) SubscribeCourse(ctx context.Context, courseID, userID uuid.UUID) error {
	now := time.Now().UTC()
	query := `
        INSERT INTO course_subscriptions (course_id, user_id, created_at)
        VALUES ($1, $2, $3)
    `
	_, err := r.db.Exec(ctx, query, courseID, userID, now)
	if err != nil {
		if pgErr := UnwrapPgError(err); pgErr != nil && pgErr.Code == "23505" {
			return app_errors.ErrAlreadySubscribed
		}
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	return nil
}

func (r *SubscriptionPostgres) GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.Course, error) {
	query := `
        SELECT c.id, c.title, c.description, c.logo_object_key, c.created_at, c.updated_at, c.author_id, c.status, c.stars_count
        FROM courses c
        INNER JOIN course_subscriptions cs ON cs.course_id = c.id
        WHERE cs.user_id = $1
        ORDER BY c.created_at DESC
    `
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscribed courses: %w", err)
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.LogoObjectKey, &c.CreatedAt, &c.UpdatedAt, &c.AuthorID, &c.Status, &c.StarsCount); err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	return courses, nil
}
