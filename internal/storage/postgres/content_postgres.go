package postgres

import (
	"SkillForge/internal/models"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type ContentPostgres struct {
	db *pgxpool.Pool
}

func NewContentPostgres(db *pgxpool.Pool) *ContentPostgres {
	return &ContentPostgres{db: db}
}

func (r *ContentPostgres) CreateContent(ctx context.Context, c models.CourseContent) (models.CourseContent, error) {
	query := `
    INSERT INTO contents (
        id, lesson_id, type, order_num,
        text, object_key, quiz_json, created_at, updated_at
    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
    `
	now := time.Now().UTC()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	c.CreatedAt = now
	c.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		c.ID, c.LessonID, c.Type, c.Order,
		c.Text, c.ObjectKey, c.QuizJSON,
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return models.CourseContent{}, fmt.Errorf("failed to insert content: %w", err)
	}
	return c, nil
}

func (r *ContentPostgres) GetContentsByLesson(ctx context.Context, lessonID uuid.UUID) ([]models.CourseContent, error) {
	query := `
    SELECT id, lesson_id, type, order_num, text, object_key, quiz_json, created_at, updated_at
      FROM contents
     WHERE lesson_id = $1
  ORDER BY order_num
    `
	rows, err := r.db.Query(ctx, query, lessonID)
	if err != nil {
		return nil, fmt.Errorf("failed to query contents: %w", err)
	}
	defer rows.Close()

	var contents []models.CourseContent
	for rows.Next() {
		var c models.CourseContent
		if err := rows.Scan(
			&c.ID, &c.LessonID, &c.Type, &c.Order,
			&c.Text, &c.ObjectKey, &c.QuizJSON,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contents = append(contents, c)
	}
	return contents, nil
}

func (r *ContentPostgres) UpdateContent(ctx context.Context, c models.CourseContent) (models.CourseContent, error) {
	query := `
    UPDATE contents SET
        type = $1,
        order_num = $2,
        text = $3,
        object_key = $4,
        quiz_json = $5,
        updated_at = $6
     WHERE id = $7
    `
	c.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, query,
		c.Type, c.Order, c.Text, c.ObjectKey, c.QuizJSON, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return models.CourseContent{}, fmt.Errorf("failed to update content: %w", err)
	}
	return c, nil
}

func (r *ContentPostgres) DeleteContent(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM contents WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}
	return nil
}
