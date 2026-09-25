ALTER TABLE users ADD COLUMN IF NOT EXISTS email text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
DO $$
BEGIN
 IF NOT EXISTS (
  SELECT 1 FROM pg_attrdef d
  JOIN pg_attribute a ON a.attrelid=d.adrelid AND a.attnum=d.adnum
  WHERE d.adrelid='users'::regclass AND a.attname='id'
 ) THEN
  CREATE SEQUENCE IF NOT EXISTS users_id_seq;
  PERFORM setval('users_id_seq', COALESCE((SELECT max(id) FROM users),0)+1, false);
  ALTER TABLE users ALTER COLUMN id SET DEFAULT nextval('users_id_seq');
  ALTER SEQUENCE users_id_seq OWNED BY users.id;
 END IF;
END $$;
UPDATE users SET email='demo@example.com' WHERE id=1 AND email IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_users_email ON users(lower(email)) WHERE email IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_roles (
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 role text NOT NULL CHECK(role IN ('resident','manager','admin')),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(user_id, role)
);
INSERT INTO user_roles(user_id,role)
SELECT id,'resident' FROM users
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS user_organizations (
 user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 organization_id bigint NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_login_codes (
 id bigserial PRIMARY KEY,
 email text NOT NULL,
 code_hash text NOT NULL,
 expires_at timestamptz NOT NULL,
 consumed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS auth_login_codes_email_idx ON auth_login_codes(lower(email), created_at DESC);

CREATE TABLE IF NOT EXISTS auth_sessions (
 token_hash text PRIMARY KEY,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS auth_sessions_user_idx ON auth_sessions(user_id);

CREATE TABLE IF NOT EXISTS user_apartments (
 id bigserial PRIMARY KEY,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 house_object_id bigint NOT NULL,
 house_object_guid uuid,
 apartment_object_id bigint,
 apartment_object_guid uuid,
 address text NOT NULL,
 label text NOT NULL DEFAULT '',
 is_default boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS user_apartments_user_idx ON user_apartments(user_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_default_apartment ON user_apartments(user_id) WHERE is_default;
