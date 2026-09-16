ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_kind_check;
ALTER TABLE requests ADD CONSTRAINT requests_kind_check CHECK(kind IN ('PROBLEM','APPLICATION','QUESTION','EMERGENCY','COMPLAINT'));
ALTER TABLE request_status_history ADD COLUMN IF NOT EXISTS comment text NOT NULL DEFAULT '';
ALTER TABLE request_status_history ADD COLUMN IF NOT EXISTS actor text NOT NULL DEFAULT 'Житель';
ALTER TABLE houses ADD COLUMN IF NOT EXISTS total_area numeric NOT NULL DEFAULT 0;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS living_area numeric NOT NULL DEFAULT 0;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS floors integer NOT NULL DEFAULT 0;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS entrances integer NOT NULL DEFAULT 0;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS apartments integer NOT NULL DEFAULT 0;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS year_built integer NOT NULL DEFAULT 0;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS organization_id bigint REFERENCES organizations(id);
ALTER TABLE houses ADD COLUMN IF NOT EXISTS manager text NOT NULL DEFAULT '';
ALTER TABLE houses ADD COLUMN IF NOT EXISTS contact text NOT NULL DEFAULT '';
UPDATE houses SET total_area=12480,living_area=9360,floors=9,entrances=4,apartments=144,year_built=1987,
 organization_id=1,manager='Анна Смирнова · демонстрационный ответственный',contact='Контакты диспетчерской пока не подключены'
 WHERE address='г. Москва, ул. Тестовая, д. 1';
