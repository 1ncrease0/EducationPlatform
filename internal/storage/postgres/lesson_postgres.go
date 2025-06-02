package postgres

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LessonPostgres struct {
	db *pgxpool.Pool
}

func NewLessonPostgres(db *pgxpool.Pool) *LessonPostgres {
	return &LessonPostgres{db: db}
}

func (r *LessonPostgres) CreateLesson(ctx context.Context, lesson models.Lesson) (*models.Lesson, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	updateQuery := `
        UPDATE lessons SET lesson_order = lesson_order + 1
         WHERE module_id = $1 AND lesson_order >= $2
    `
	_, err = tx.Exec(ctx, updateQuery, lesson.ModuleID, lesson.LessonOrder)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if lesson.ID == uuid.Nil {
		lesson.ID = uuid.New()
	}
	lesson.CreatedAt = now
	lesson.UpdatedAt = now

	insertQuery := `
    INSERT INTO lessons (
        id, course_id, module_id,
        lesson_title, lesson_order, created_at, updated_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
	_, err = tx.Exec(ctx, insertQuery,
		lesson.ID, lesson.CourseID, lesson.ModuleID,
		lesson.LessonTitle, lesson.LessonOrder, lesson.CreatedAt, lesson.UpdatedAt,
	)
	if err != nil {
		if pgErr := UnwrapPgError(err); pgErr != nil && pgErr.Code == "23505" {
			return nil, app_errors.ErrDuplicateLesson
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (r *LessonPostgres) GetLessonByID(ctx context.Context, id uuid.UUID) (models.Lesson, error) {
	var lesson models.Lesson
	query := `
    SELECT id, course_id, module_id,
           lesson_title, lesson_order, created_at, updated_at
      FROM lessons
     WHERE id = $1
    `
	row := r.db.QueryRow(ctx, query, id)
	err := row.Scan(
		&lesson.ID, &lesson.CourseID, &lesson.ModuleID,
		&lesson.LessonTitle, &lesson.LessonOrder, &lesson.CreatedAt, &lesson.UpdatedAt,
	)
	if err != nil {
		return models.Lesson{}, fmt.Errorf("lesson not found: %w", err)
	}
	return lesson, nil
}

func (r *LessonPostgres) GetMaxLessonOrder(ctx context.Context, moduleID uuid.UUID) (int, error) {
	var max int
	query := `SELECT COALESCE(MAX(lesson_order), 0) FROM lessons WHERE module_id = $1`
	err := r.db.QueryRow(ctx, query, moduleID).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("failed to get max lesson order: %w", err)
	}
	return max, nil
}

func (r *LessonPostgres) CreateModule(ctx context.Context, module models.Module) (*models.Module, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Сдвигаем order для модулей, если нужно
	updateQuery := `
        UPDATE modules SET module_order = module_order + 1
         WHERE course_id = $1 AND module_order >= $2
    `
	_, err = tx.Exec(ctx, updateQuery, module.CourseID, module.Order)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if module.ID == uuid.Nil {
		module.ID = uuid.New()
	}
	module.CreatedAt = now
	module.UpdatedAt = now

	insertQuery := `
    INSERT INTO modules (
        id, course_id, title, module_order, created_at, updated_at
    ) VALUES ($1, $2, $3, $4, $5, $6)
    `
	_, err = tx.Exec(ctx, insertQuery,
		module.ID, module.CourseID, module.Title,
		module.Order, module.CreatedAt, module.UpdatedAt,
	)
	if err != nil {
		if pgErr := UnwrapPgError(err); pgErr != nil && pgErr.Code == "23505" {
			return nil, app_errors.ErrDuplicateModule
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &module, nil
}

func (r *LessonPostgres) GetMaxModuleOrder(ctx context.Context, courseID uuid.UUID) (int, error) {
	var max int
	query := `SELECT COALESCE(MAX(module_order), 0) FROM modules WHERE course_id = $1`
	err := r.db.QueryRow(ctx, query, courseID).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("failed to get max module order: %w", err)
	}
	return max, nil
}

func (r *LessonPostgres) DeleteLessonAndUpdateOrder(ctx context.Context, lessonID, moduleID uuid.UUID, lessonOrder int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	deleteQuery := `DELETE FROM lessons WHERE id = $1`
	_, err = tx.Exec(ctx, deleteQuery, lessonID)
	if err != nil {
		return err
	}

	updateQuery := `
        UPDATE lessons SET lesson_order = lesson_order - 1
         WHERE module_id = $1 AND lesson_order > $2
    `
	_, err = tx.Exec(ctx, updateQuery, moduleID, lessonOrder)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *LessonPostgres) DeleteModuleAndUpdateOrder(ctx context.Context, moduleID, courseID uuid.UUID, moduleOrder int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	deleteLessonsQuery := `DELETE FROM lessons WHERE module_id = $1`
	_, err = tx.Exec(ctx, deleteLessonsQuery, moduleID)
	if err != nil {
		return err
	}

	deleteModuleQuery := `DELETE FROM modules WHERE id = $1`
	_, err = tx.Exec(ctx, deleteModuleQuery, moduleID)
	if err != nil {
		return err
	}

	updateQuery := `
        UPDATE modules SET module_order = module_order - 1
         WHERE course_id = $1 AND module_order > $2
    `
	_, err = tx.Exec(ctx, updateQuery, courseID, moduleOrder)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *LessonPostgres) CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error) {
	modulesQuery := `
        SELECT id, course_id, title, module_order, created_at, updated_at
        FROM modules
        WHERE course_id = $1
        ORDER BY module_order
    `
	rows, err := r.db.Query(ctx, modulesQuery, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query modules: %w", err)
	}
	defer rows.Close()

	var modules []models.Module
	for rows.Next() {
		var m models.Module
		if err := rows.Scan(&m.ID, &m.CourseID, &m.Title, &m.Order, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}

	lessonsQuery := `
        SELECT id, course_id, module_id, lesson_title, lesson_order, created_at, updated_at
        FROM lessons
        WHERE course_id = $1
        ORDER BY module_id, lesson_order
    `
	lessonRows, err := r.db.Query(ctx, lessonsQuery, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lessons: %w", err)
	}
	defer lessonRows.Close()

	lessonsByModule := make(map[uuid.UUID][]models.Lesson)
	for lessonRows.Next() {
		var l models.Lesson
		if err := lessonRows.Scan(&l.ID, &l.CourseID, &l.ModuleID, &l.LessonTitle, &l.LessonOrder, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		lessonsByModule[l.ModuleID] = append(lessonsByModule[l.ModuleID], l)
	}

	var content []models.Contents
	for _, mod := range modules {
		ml := models.Contents{
			Module:  mod,
			Lessons: lessonsByModule[mod.ID],
		}
		content = append(content, ml)
	}
	return content, nil
}

func (r *LessonPostgres) GetModuleByID(ctx context.Context, moduleID uuid.UUID) (models.Module, error) {
	var module models.Module
	query := `
        SELECT id, course_id, title, module_order, created_at, updated_at
          FROM modules
         WHERE id = $1
    `
	row := r.db.QueryRow(ctx, query, moduleID)
	err := row.Scan(&module.ID, &module.CourseID, &module.Title, &module.Order, &module.CreatedAt, &module.UpdatedAt)
	if err != nil {
		return models.Module{}, fmt.Errorf("module not found: %w", err)
	}
	return module, nil
}

func (r *LessonPostgres) LessonsByCourse(ctx context.Context, courseID uuid.UUID) ([]models.Lesson, error) {
	query := `
        SELECT id, course_id, module_id, lesson_title, lesson_order, created_at, updated_at
          FROM lessons
         WHERE course_id = $1
         ORDER BY lesson_order
    `
	rows, err := r.db.Query(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lessons by course: %w", err)
	}
	defer rows.Close()

	var lessons []models.Lesson
	for rows.Next() {
		var l models.Lesson
		if err := rows.Scan(
			&l.ID, &l.CourseID, &l.ModuleID, &l.LessonTitle, &l.LessonOrder, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		lessons = append(lessons, l)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return lessons, nil
}
func (r *LessonPostgres) SwapLessons(ctx context.Context, lessonID1, lessonID2 uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var order1, order2 int
	query := `SELECT lesson_order FROM lessons WHERE id = $1`
	if err := tx.QueryRow(ctx, query, lessonID1).Scan(&order1); err != nil {
		return fmt.Errorf("failed to get order for lesson1: %w", err)
	}
	if err := tx.QueryRow(ctx, query, lessonID2).Scan(&order2); err != nil {
		return fmt.Errorf("failed to get order for lesson2: %w", err)
	}

	updateQuery := `UPDATE lessons SET lesson_order = $1 WHERE id = $2`
	tempOrder := -1
	if _, err := tx.Exec(ctx, updateQuery, tempOrder, lessonID1); err != nil {
		return fmt.Errorf("failed to update lesson1 to temp order: %w", err)
	}
	if _, err := tx.Exec(ctx, updateQuery, order1, lessonID2); err != nil {
		return fmt.Errorf("failed to update lesson2 order: %w", err)
	}
	if _, err := tx.Exec(ctx, updateQuery, order2, lessonID1); err != nil {
		return fmt.Errorf("failed to update lesson1 order: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *LessonPostgres) SwapModules(ctx context.Context, moduleID1, moduleID2 uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var order1, order2 int
	query := `SELECT module_order FROM modules WHERE id = $1`
	if err := tx.QueryRow(ctx, query, moduleID1).Scan(&order1); err != nil {
		return fmt.Errorf("failed to get order for module1: %w", err)
	}
	if err := tx.QueryRow(ctx, query, moduleID2).Scan(&order2); err != nil {
		return fmt.Errorf("failed to get order for module2: %w", err)
	}

	updateQuery := `UPDATE modules SET module_order = $1 WHERE id = $2`
	tempOrder := -1
	if _, err := tx.Exec(ctx, updateQuery, tempOrder, moduleID1); err != nil {
		return fmt.Errorf("failed to update module1 to temp order: %w", err)
	}
	if _, err := tx.Exec(ctx, updateQuery, order1, moduleID2); err != nil {
		return fmt.Errorf("failed to update module2 order: %w", err)
	}
	if _, err := tx.Exec(ctx, updateQuery, order2, moduleID1); err != nil {
		return fmt.Errorf("failed to update module1 order: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *LessonPostgres) CreateContent(ctx context.Context, content models.CourseContent) (*models.CourseContent, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var max int
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(order_num), 0) FROM contents WHERE lesson_id = $1`, content.LessonID).Scan(&max)
	if err != nil {
		return nil, fmt.Errorf("failed to get max order_num: %w", err)
	}
	content.Order = max + 1

	now := time.Now().UTC()
	content.CreatedAt = now
	content.UpdatedAt = now
	if content.ID == uuid.Nil {
		content.ID = uuid.New()
	}

	insertQuery := `
    INSERT INTO contents (
        id, lesson_id, type, order_num, text, object_key, quiz_json, created_at, updated_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
	_, err = tx.Exec(ctx, insertQuery,
		content.ID, content.LessonID, content.Type, content.Order,
		content.Text, content.ObjectKey, content.QuizJSON,
		content.CreatedAt, content.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &content, nil
}

func (r *LessonPostgres) GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error) {
	var detail models.LessonDetail
	query := `
        SELECT id, course_id, module_id, lesson_title, lesson_order, created_at, updated_at 
          FROM lessons 
         WHERE id = $1
    `
	row := r.db.QueryRow(ctx, query, lessonID)
	if err := row.Scan(&detail.Lesson.ID, &detail.Lesson.CourseID, &detail.Lesson.ModuleID, &detail.Lesson.LessonTitle, &detail.Lesson.LessonOrder, &detail.Lesson.CreatedAt, &detail.Lesson.UpdatedAt); err != nil {
		return detail, fmt.Errorf("lesson not found: %w", err)
	}
	contentsQuery := `
        SELECT id, lesson_id, type, order_num, text, object_key, quiz_json, created_at, updated_at 
          FROM contents 
         WHERE lesson_id = $1 
         ORDER BY order_num
    `
	rows, err := r.db.Query(ctx, contentsQuery, lessonID)
	if err != nil {
		return detail, fmt.Errorf("failed to query contents: %w", err)
	}
	defer rows.Close()

	var contents []models.CourseContent
	for rows.Next() {
		var c models.CourseContent
		if err := rows.Scan(&c.ID, &c.LessonID, &c.Type, &c.Order, &c.Text, &c.ObjectKey, &c.QuizJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return detail, err
		}
		contents = append(contents, c)
	}
	detail.Contents = contents
	return detail, nil
}

func (r *LessonPostgres) LessonsByModule(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error) {
	query := `
        SELECT id, course_id, module_id, lesson_title, lesson_order, created_at, updated_at
          FROM lessons
         WHERE module_id = $1
         ORDER BY lesson_order
    `
	rows, err := r.db.Query(ctx, query, moduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lessons by module: %w", err)
	}
	defer rows.Close()

	var lessons []models.Lesson
	for rows.Next() {
		var lesson models.Lesson
		if err := rows.Scan(&lesson.ID, &lesson.CourseID, &lesson.ModuleID, &lesson.LessonTitle, &lesson.LessonOrder, &lesson.CreatedAt, &lesson.UpdatedAt); err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return lessons, nil
}

func (r *LessonPostgres) UpsertContent(ctx context.Context, content models.CourseContent) (*models.CourseContent, error) {
	var existingID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM contents WHERE lesson_id = $1 LIMIT 1`, content.LessonID).Scan(&existingID)
	now := time.Now().UTC()
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "pg: no rows in result set" {
			content.Order = 1
			content.CreatedAt = now
			content.UpdatedAt = now
			if content.ID == uuid.Nil {
				content.ID = uuid.New()
			}
			insertQuery := `
                INSERT INTO contents (
                    id, lesson_id, type, order_num, text, object_key, quiz_json, created_at, updated_at
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            `
			_, err = r.db.Exec(ctx, insertQuery,
				content.ID, content.LessonID, content.Type, content.Order,
				content.Text, content.ObjectKey, content.QuizJSON,
				content.CreatedAt, content.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
			return &content, nil
		} else {
			return nil, err
		}
	}
	updateQuery := `
        UPDATE contents SET type = $1, text = $2, object_key = $3, quiz_json = $4, updated_at = $5
        WHERE lesson_id = $6
    `
	_, err = r.db.Exec(ctx, updateQuery,
		content.Type, content.Text, content.ObjectKey, content.QuizJSON,
		now, content.LessonID,
	)
	if err != nil {
		return nil, err
	}
	content.ID = existingID
	content.UpdatedAt = now
	return &content, nil
}

func (r *LessonPostgres) UpdateLessonProgress(ctx context.Context, lessonID, userID uuid.UUID, status string, score float64) error {
	query := `
		INSERT INTO lesson_progress (user_id, lesson_id, status, score, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, lesson_id) 
		DO UPDATE SET status = $3, score = $4, updated_at = $5
	`
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, query, userID, lessonID, status, score, now)
	if err != nil {
		return fmt.Errorf("failed to update lesson progress: %w", err)
	}
	return nil
}

func (r *LessonPostgres) GetLessonProgress(ctx context.Context, lessonID, userID uuid.UUID) (models.LessonProgress, error) {
	query := `
		SELECT user_id, lesson_id, status, score, updated_at
		FROM lesson_progress
		WHERE user_id = $1 AND lesson_id = $2
	`
	var progress models.LessonProgress
	err := r.db.QueryRow(ctx, query, userID, lessonID).Scan(
		&progress.UserID,
		&progress.LessonID,
		&progress.Status,
		&progress.Score,
		&progress.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.LessonProgress{
				UserID:    userID,
				LessonID:  lessonID,
				Status:    models.LessonStatusFailed,
				Score:     0,
				UpdatedAt: time.Now().UTC(),
			}, nil
		}
		return models.LessonProgress{}, fmt.Errorf("failed to get lesson progress: %w", err)
	}
	return progress, nil
}
