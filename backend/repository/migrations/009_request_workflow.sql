CREATE TABLE IF NOT EXISTS organization_types (
 code text PRIMARY KEY,
 name text NOT NULL
);
INSERT INTO organization_types(code,name) VALUES
 ('MANAGEMENT_COMPANY','Управляющая организация'),
 ('HOA_COOPERATIVE','ТСЖ или кооператив'),
 ('RESOURCE_SUPPLIER','Ресурсоснабжающая организация'),
 ('CONTRACTOR','Подрядная организация'),
 ('REGIONAL_OPERATOR','Региональный оператор'),
 ('MUNICIPAL','Муниципальная организация'),
 ('HOUSING_INSPECTION','Жилищная инспекция'),
 ('OTHER','Другой тип')
ON CONFLICT DO NOTHING;

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS type_code text REFERENCES organization_types(code);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS inn text;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS ogrn text;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS external_guid text;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS phone text NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS website text NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS source_url text NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS source_snapshot_at timestamptz;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT true;
UPDATE organizations SET type_code='MANAGEMENT_COMPANY' WHERE type_code IS NULL;
ALTER TABLE organizations ALTER COLUMN type_code SET NOT NULL;
ALTER TABLE organizations ALTER COLUMN type_code SET DEFAULT 'MANAGEMENT_COMPANY';
CREATE UNIQUE INDEX IF NOT EXISTS ux_organizations_external_guid ON organizations(external_guid) WHERE external_guid IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_organizations_ogrn ON organizations(ogrn) WHERE ogrn IS NOT NULL;

CREATE TABLE IF NOT EXISTS house_responsibility_rules (
 id bigserial PRIMARY KEY,
 house_id bigint NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
 organization_id bigint NOT NULL REFERENCES organizations(id),
 category text NOT NULL,
 place text NOT NULL DEFAULT '',
 urgency text NOT NULL DEFAULT '',
 organization_role text NOT NULL CHECK(organization_role IN ('PRIMARY','CONTRACTOR','ESCALATION')),
 valid_from timestamptz NOT NULL,
 valid_to timestamptz,
 active boolean NOT NULL DEFAULT true,
 source text NOT NULL,
 source_url text NOT NULL DEFAULT '',
 is_demo boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK(valid_to IS NULL OR valid_from <= valid_to)
);
CREATE INDEX IF NOT EXISTS responsibility_rules_lookup_idx ON house_responsibility_rules(house_id,category,active,valid_from,valid_to);

ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_problem_type_check;
ALTER TABLE requests ADD CONSTRAINT requests_problem_type_check CHECK(problem_type IN ('PIPE_LEAK','ELEVATOR','HEATING','WATER_SUPPLY','ELECTRICITY','WASTE','ROOF','CAPITAL_REPAIR','OUTDOOR_LIGHTING','OTHER'));

ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_status_check;
ALTER TABLE requests ADD CONSTRAINT requests_status_check CHECK(status IN ('ROUTING_REQUIRED','CREATED','SENT','ACCEPTED','IN_PROGRESS','RESOLVED','CLOSED','REJECTED'));
ALTER TABLE request_status_history DROP CONSTRAINT IF EXISTS request_status_history_status_check;
ALTER TABLE request_status_history ADD CONSTRAINT request_status_history_status_check CHECK(status IN ('ROUTING_REQUIRED','CREATED','SENT','ACCEPTED','IN_PROGRESS','RESOLVED','CLOSED','REJECTED'));

ALTER TABLE requests ADD COLUMN IF NOT EXISTS problem_place text NOT NULL DEFAULT 'COMMON_PROPERTY';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS urgency text NOT NULL DEFAULT 'NORMAL';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS primary_organization_id bigint REFERENCES organizations(id);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS contractor_organization_id bigint REFERENCES organizations(id);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS routing_rule_id bigint REFERENCES house_responsibility_rules(id);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS primary_organization_name_snapshot text NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS contractor_organization_name_snapshot text NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS routing_reason text NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS routing_source text NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS assigned_user_id bigint REFERENCES users(id);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS visit_start timestamptz;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS visit_end timestamptz;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS executor_contact text NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS final_report text NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS awaiting_party text NOT NULL DEFAULT 'NONE' CHECK(awaiting_party IN ('NONE','RESIDENT','ORGANIZATION'));
ALTER TABLE requests ADD COLUMN IF NOT EXISTS reopen_count integer NOT NULL DEFAULT 0 CHECK(reopen_count >= 0);
ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_visit_interval_check;
ALTER TABLE requests ADD CONSTRAINT requests_visit_interval_check CHECK(visit_start IS NULL OR visit_end IS NULL OR visit_start <= visit_end);
UPDATE requests r SET
 primary_organization_id=COALESCE(primary_organization_id,responsible_organization_id),
 primary_organization_name_snapshot=CASE WHEN primary_organization_name_snapshot='' THEN o.name ELSE primary_organization_name_snapshot END,
 routing_reason=CASE WHEN routing_reason='' THEN 'Сохранено из прежней версии заявки' ELSE routing_reason END,
 routing_source=CASE WHEN routing_source='' THEN 'legacy' ELSE routing_source END
FROM organizations o WHERE o.id=r.responsible_organization_id;
ALTER TABLE requests ALTER COLUMN responsible_organization_id DROP NOT NULL;

CREATE TABLE IF NOT EXISTS request_resolution_feedback (
 id bigserial PRIMARY KEY,
 request_id bigint NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
 solved boolean NOT NULL,
 rating integer CHECK(rating BETWEEN 1 AND 5),
 comment text NOT NULL DEFAULT '',
 author_user_id bigint NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS resolution_feedback_request_idx ON request_resolution_feedback(request_id,id);

CREATE TABLE IF NOT EXISTS request_messages (
 id bigserial PRIMARY KEY,
 request_id bigint NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
 message_type text NOT NULL CHECK(message_type IN ('RESIDENT_PUBLIC','ORGANIZATION_PUBLIC','INTERNAL_NOTE','SYSTEM_EVENT','INFO_REQUEST')),
 message_text text NOT NULL DEFAULT '',
 author_user_id bigint REFERENCES users(id),
 author_name text NOT NULL,
 author_role text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS request_messages_request_idx ON request_messages(request_id,id);

CREATE TABLE IF NOT EXISTS request_message_attachments (
 id bigserial PRIMARY KEY,
 message_id bigint NOT NULL REFERENCES request_messages(id) ON DELETE CASCADE,
 original_name text NOT NULL,
 mime_type text NOT NULL CHECK(mime_type IN ('image/jpeg','image/png','image/webp')),
 byte_size integer NOT NULL CHECK(byte_size > 0 AND byte_size <= 5242880),
 content bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS request_attachments_message_idx ON request_message_attachments(message_id,id);

ALTER TABLE request_status_history ADD COLUMN IF NOT EXISTS actor_user_id bigint REFERENCES users(id);
ALTER TABLE request_status_history ADD COLUMN IF NOT EXISTS actor_role text NOT NULL DEFAULT '';
