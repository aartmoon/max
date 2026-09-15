CREATE TABLE IF NOT EXISTS users (id bigint PRIMARY KEY, name text NOT NULL);
CREATE TABLE IF NOT EXISTS houses (id bigserial PRIMARY KEY, address text NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS organizations (id bigint PRIMARY KEY, name text NOT NULL);
CREATE TABLE IF NOT EXISTS requests (
 id bigserial PRIMARY KEY,
 user_id bigint NOT NULL REFERENCES users(id),
 house_id bigint NOT NULL REFERENCES houses(id),
 description text NOT NULL,
 problem_type text NOT NULL CHECK(problem_type IN ('PIPE_LEAK','ELEVATOR','HEATING','OTHER')),
 responsible_organization_id bigint NOT NULL REFERENCES organizations(id),
 status text NOT NULL CHECK(status IN ('CREATED','SENT','ACCEPTED','IN_PROGRESS','RESOLVED','REJECTED')),
 deadline timestamptz NOT NULL,
 created_at timestamptz NOT NULL,
 kind text NOT NULL CHECK(kind IN ('PROBLEM','APPLICATION','QUESTION')),
 request_text text NOT NULL,
 photo bytea,
 photo_type text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS requests_user_idx ON requests(user_id, created_at DESC);
CREATE TABLE IF NOT EXISTS request_status_history (
 id bigserial PRIMARY KEY,
 request_id bigint NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
 status text NOT NULL CHECK(status IN ('CREATED','SENT','ACCEPTED','IN_PROGRESS','RESOLVED','REJECTED')),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS history_request_idx ON request_status_history(request_id,id);
INSERT INTO users VALUES (1,'Тестовый житель') ON CONFLICT DO NOTHING;
INSERT INTO houses(address) VALUES ('г. Москва, ул. Тестовая, д. 1') ON CONFLICT DO NOTHING;
INSERT INTO organizations VALUES (1,'УК «Тестовая»'),(2,'УК «Тестовая» / подрядчик «ТестЛифт»') ON CONFLICT DO NOTHING;
