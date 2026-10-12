CREATE TABLE IF NOT EXISTS boards (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS threads (
    id BIGSERIAL PRIMARY KEY,
    board_id BIGINT NOT NULL REFERENCES boards(id),
    title TEXT NOT NULL,
    owner TEXT NOT NULL DEFAULT '',
    owner_only BOOLEAN NOT NULL DEFAULT FALSE,
    post_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    last_posted_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS posts (
    id BIGSERIAL PRIMARY KEY,
    thread_id BIGINT NOT NULL REFERENCES threads(id),
    number INTEGER NOT NULL,
    author TEXT NOT NULL,
    body TEXT NOT NULL,
    sage BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE(thread_id, number)
);
INSERT INTO boards (name, description, created_at) VALUES
    ('programming', 'Programming General', NOW()),
    ('news', 'News & Current Events', NOW()),
    ('chat', 'Casual Chat', NOW())
ON CONFLICT(name) DO NOTHING;
