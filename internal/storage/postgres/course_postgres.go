package postgres

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type CoursePostgres struct {
	db *pgxpool.Pool
}

func NewCoursePostgres(db *pgxpool.Pool) *CoursePostgres {
	return &CoursePostgres{db: db}
}

func (r *CoursePostgres) NewCourse(ctx context.Context, course *models.Course) (uuid.UUID, error) {
	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}
	now := time.Now().UTC()
	course.CreatedAt = now
	course.UpdatedAt = now
	query := `
		INSERT INTO courses (
			id, title, description, image_url, created_at, updated_at,
			author_id, status, stars_count
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9
		)
		RETURNING id, created_at, updated_at
	`
	var returnedID uuid.UUID
	var returnedCreated, returnedUpdated time.Time
	err := r.db.QueryRow(
		ctx,
		query,
		course.ID,
		course.Title,
		course.Description,
		course.ImageURL,
		course.CreatedAt,
		course.UpdatedAt,
		course.AuthorID,
		course.Status,
		course.StarsCount,
	).Scan(&returnedID, &returnedCreated, &returnedUpdated)
	if err != nil {
		return uuid.Nil, err
	}
	course.ID = returnedID
	course.CreatedAt = returnedCreated
	course.UpdatedAt = returnedUpdated
	return returnedID, nil
}

func (r *CoursePostgres) CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error) {
	const query = `
        SELECT 
            id,
            title,
            description,
            image_url,
            created_at,
            updated_at,
            author_id,
            status,
            stars_count
        FROM courses
        WHERE id = $1
    `
	course := &models.Course{}
	row := r.db.QueryRow(ctx, query, id)
	err := row.Scan(
		&course.ID,
		&course.Title,
		&course.Description,
		&course.ImageURL,
		&course.CreatedAt,
		&course.UpdatedAt,
		&course.AuthorID,
		&course.Status,
		&course.StarsCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, app_errors.ErrCourseNotFound
		}
		return nil, err
	}

	return course, nil
}

func (r *CoursePostgres) ChangeStatus(ctx context.Context, id uuid.UUID, status string) error {
	const query = `
        UPDATE courses
           SET status     = $2,
               updated_at = NOW()
         WHERE id = $1
    `
	cmdTag, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return app_errors.ErrCourseNotFound
	}
	return nil
}
