ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_problem_type_check;
ALTER TABLE requests ADD CONSTRAINT requests_problem_type_check CHECK(problem_type IN ('PIPE_LEAK','ELEVATOR','HEATING','WATER_SUPPLY','ELECTRICITY','WASTE','ROOF','CAPITAL_REPAIR','OUTDOOR_LIGHTING','OTHER'));
ALTER TABLE organizations ALTER COLUMN type_code SET DEFAULT 'MANAGEMENT_COMPANY';

INSERT INTO organizations(id,name,type_code,phone,website,source,source_url,source_snapshot_at,active) VALUES
 (1009,'Фонд капитального ремонта многоквартирных домов города Москвы','REGIONAL_OPERATOR','','https://fond.mos.ru','Официальные документы Правительства Москвы','https://www.mos.ru/upload/documents/oiv/ad_kr_605.pdf',TIMESTAMPTZ '2026-09-29 00:00:00+00',true)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,type_code=EXCLUDED.type_code,source=EXCLUDED.source,source_url=EXCLUDED.source_url,source_snapshot_at=EXCLUDED.source_snapshot_at;

INSERT INTO house_responsibility_rules(house_id,organization_id,category,place,urgency,organization_role,valid_from,source,source_url,is_demo)
SELECT h.id,1009,'CAPITAL_REPAIR','','','PRIMARY',TIMESTAMPTZ '2026-09-29 00:00:00+00','Демонстрационное правило на основе публичных данных Арбата','https://www.mos.ru/upload/documents/oiv/ad_kr_605.pdf',true
FROM houses h
WHERE h.address ILIKE '%Арбат%'
AND NOT EXISTS (
 SELECT 1 FROM house_responsibility_rules r
 WHERE r.house_id=h.id AND r.organization_id=1009 AND r.category='CAPITAL_REPAIR' AND r.place='' AND r.organization_role='PRIMARY' AND r.is_demo
);
