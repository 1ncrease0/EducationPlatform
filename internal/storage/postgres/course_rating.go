package postgres

import (
	"SkillForge/internal/app_errors"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseRatingPostgres struct {
	db *pgxpool.Pool
}

func NewCourseRatingPostgres(db *pgxpool.Pool) *CourseRatingPostgres {
	return &CourseRatingPostgres{db: db}
}

func (r *CourseRatingPostgres) AddRating(ctx context.Context, courseID, userID uuid.UUID) error {
	var exists bool
	err := r.db.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM course_ratings WHERE course_id=$1 AND user_id=$2
        )
    `, courseID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return app_errors.ErrAlreadyRated
	}
	_, err = r.db.Exec(ctx, `
        INSERT INTO course_ratings (course_id, user_id)
        VALUES ($1, $2)
    `, courseID, userID)
	return err
}

func (r *CourseRatingPostgres) RemoveRating(ctx context.Context, courseID, userID uuid.UUID) error {
	var exists bool
	err := r.db.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM course_ratings WHERE course_id=$1 AND user_id=$2
        )
    `, courseID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return app_errors.ErrNotRated
	}
	_, err = r.db.Exec(ctx, `
        DELETE FROM course_ratings WHERE course_id=$1 AND user_id=$2
    `, courseID, userID)
	return err
}

func (r *CourseRatingPostgres) IsRated(ctx context.Context, courseID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM course_ratings WHERE course_id=$1 AND user_id=$2
        )
    `, courseID, userID).Scan(&exists)
	return exists, err
}
