CREATE TABLE IF NOT EXISTS user_max_accounts (
 user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 max_user_id bigint NOT NULL UNIQUE,
 chat_id bigint NOT NULL,
 linked_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);

