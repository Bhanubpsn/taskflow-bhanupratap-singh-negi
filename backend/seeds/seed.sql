-- Idempotent seed: safe to run multiple times.
-- Password for test@example.com is: password123  (bcrypt cost 12)

DO $$
DECLARE
    v_user_id    UUID;
    v_project_id UUID;
BEGIN

    -- ── User ─────────────────────────────────────────────────────────────────
    INSERT INTO users (name, email, password)
    VALUES (
        'Test User',
        'test@example.com',
        '$2a$12$92NgBaONZWY3e37B5Znln..8Ml4GR5uzmknbpDjmvxjLKRpRKd1JK'
    )
    ON CONFLICT (email) DO NOTHING;

    SELECT id INTO v_user_id FROM users WHERE email = 'test@example.com';

    -- ── Project ───────────────────────────────────────────────────────────────
    INSERT INTO projects (name, description, owner_id)
    VALUES (
        'TaskFlow Backend',
        'REST API for task and project management built with Go.',
        v_user_id
    )
    ON CONFLICT DO NOTHING
    RETURNING id INTO v_project_id;

    -- If the project already existed, look it up instead
    IF v_project_id IS NULL THEN
        SELECT id INTO v_project_id
        FROM projects
        WHERE name = 'TaskFlow Backend' AND owner_id = v_user_id;
    END IF;

    -- ── Tasks ─────────────────────────────────────────────────────────────────
    INSERT INTO tasks (title, description, status, priority, project_id, created_by)
    VALUES
        (
            'Set up project structure',
            'Initialise Go module, folder layout, and Docker Compose stack.',
            'done',
            'high',
            v_project_id,
            v_user_id
        ),
        (
            'Implement auth endpoints',
            'POST /auth/register and POST /auth/login with JWT and bcrypt.',
            'in_progress',
            'high',
            v_project_id,
            v_user_id
        ),
        (
            'Write API documentation',
            'Document all endpoints with request/response examples in README.',
            'todo',
            'medium',
            v_project_id,
            v_user_id
        )
    ON CONFLICT DO NOTHING;

END $$;
