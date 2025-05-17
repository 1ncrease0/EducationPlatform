package postgres

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
			id, title, description, logo_object_key, created_at, updated_at,
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
		course.LogoObjectKey,
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
            logo_object_key,
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
		&course.LogoObjectKey,
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

func (r *CoursePostgres) CountPublicCourses(ctx context.Context) (int, error) {
	const query = `
        SELECT COUNT(*) 
          FROM courses 
         WHERE status = $1
    `
	var cnt int
	err := r.db.QueryRow(ctx, query, models.StatusPublic).Scan(&cnt)
	if err != nil {
		return 0, fmt.Errorf("CountPublicCourses: %w", err)
	}
	return cnt, nil
}

func (r *CoursePostgres) ListPublicCourses(ctx context.Context, limit int, offset int) ([]models.Course, error) {
	const query = `
   SELECT 
  id, title, description, logo_object_key, created_at, updated_at,
  author_id, status, stars_count
	FROM courses
	WHERE status = $1
	ORDER BY created_at DESC
	LIMIT $2
	OFFSET $3
    `

	rows, err := r.db.Query(ctx, query, models.StatusPublic, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ListPublicCourses: unable to query courses: %w", err)
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(
			&c.ID,
			&c.Title,
			&c.Description,
			&c.LogoObjectKey,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.AuthorID,
			&c.Status,
			&c.StarsCount,
		); err != nil {
			return nil, fmt.Errorf("ListPublicCourses: scan error: %w", err)
		}
		courses = append(courses, c)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("ListPublicCourses: rows iteration error: %w", rows.Err())
	}

	return courses, nil
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

func (r *CoursePostgres) UpdateCourseLogo(ctx context.Context, courseID uuid.UUID, logoObjectKey string) error {
	const query = `
		UPDATE courses
		   SET logo_object_key = $2,
		       updated_at      = NOW()
		 WHERE id = $1
	`
	cmd, err := r.db.Exec(ctx, query, courseID, logoObjectKey)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return app_errors.ErrCourseNotFound
	}
	return nil
}

func (r *CoursePostgres) ListCoursesByAuthor(ctx context.Context, authorID uuid.UUID) ([]models.Course, error) {
	query := `
        SELECT id, title, description, logo_object_key, created_at, updated_at, author_id, status, stars_count
        FROM courses
        WHERE author_id = $1
        ORDER BY created_at DESC
    `
	rows, err := r.db.Query(ctx, query, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to query courses by author: %w", err)
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.LogoObjectKey,
			&c.CreatedAt, &c.UpdatedAt, &c.AuthorID, &c.Status, &c.StarsCount); err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	return courses, nil
}

func (r *CoursePostgres) IncrementStars(ctx context.Context, courseID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
        UPDATE courses 
           SET stars_count = stars_count + 1
         WHERE id = $1
    `, courseID)
	return err
}

func (r *CoursePostgres) DecrementStars(ctx context.Context, courseID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
        UPDATE courses 
           SET stars_count = stars_count - 1
         WHERE id = $1 AND stars_count > 0
    `, courseID)
	return err
}
