-- Восстанавливаем числовые ID вопросов в существующих тестах
UPDATE contents
SET quiz_json = (
    SELECT jsonb_build_object(
        'title', (quiz_json->>'title'),
        'description', (quiz_json->>'description'),
        'questions', (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', (question->>'id')::bigint,
                    'text', (question->>'text'),
                    'type', (question->>'type'),
                    'options', (question->'options'),
                    'required', (question->>'required')::boolean,
                    'correct_answer', (question->>'correct_answer')
                )
            )
            FROM jsonb_array_elements(quiz_json->'questions') AS question
        ),
        'min_score', (quiz_json->>'min_score')::float
    )
)
WHERE type = 'quiz' AND quiz_json IS NOT NULL; 