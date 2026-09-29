INSERT INTO organizations(id,name,type_code,ogrn,external_guid,phone,website,source,source_url,source_snapshot_at,active) VALUES
 (1001,'ООО «ФРИСЛАНД»','MANAGEMENT_COMPANY','1127746478310','02e1ba25-5069-46c1-a951-6de95f0db656','74994262553','https://frisland.ru','Публичная карточка дома ГИС ЖКХ','https://dom.gosuslugi.ru/',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1002,'АО «МОСВОДОКАНАЛ»','RESOURCE_SUPPLIER','1127747298250','e567e8f9-5b96-4565-9555-19761dae68f6','+7 (499) 763-34-34','https://www.mosvodokanal.ru','Публичная карточка дома ГИС ЖКХ','https://dom.gosuslugi.ru/',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1003,'АО «МОСЭНЕРГОСБЫТ»','RESOURCE_SUPPLIER','1057746557329','cf076f92-5a94-41cd-8e3f-9622926d8fe9','+7 (499) 550-9-550','https://www.mosenergosbyt.ru','Публичная карточка дома ГИС ЖКХ','https://dom.gosuslugi.ru/',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1004,'ПАО «МОЭК»','RESOURCE_SUPPLIER','1047796974092','b82530ad-4ab1-41f3-b2de-1aaac72a00fe','+7 (495) 587-77-88','https://www.moek.ru','Публичная карточка дома ГИС ЖКХ','https://dom.gosuslugi.ru/',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1005,'Управа района Арбат города Москвы','MUNICIPAL','1027704014964','8ec2afc7-db42-4e86-9c53-91ae64e0c1f0','+7 (499) 252-84-22','https://arbat.mos.ru','Публичная карточка дома ГИС ЖКХ','https://dom.gosuslugi.ru/',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1006,'Мосжилинспекция','HOUSING_INSPECTION',NULL,NULL,'+7 (499) 763-18-56','https://www.mos.ru/mgi/','Правительство Москвы','https://www.mos.ru/upload/documents/files/596/PolojenieoMosjilinspekcii.pdf',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1007,'ГБУ «Жилищник района Арбат»','MANAGEMENT_COMPANY','5147746267906',NULL,'','https://arbat.mos.ru','Официальные документы Правительства Москвы','https://www.mos.ru/',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1008,'АО «Экотехпром»','REGIONAL_OPERATOR','1237700798719',NULL,'','https://eco-pro.ru','Правительство Москвы: региональный оператор ТКО на 2026–2029 годы','https://www.mos.ru/upload/documents/files/9355/ProtokolDPR-P-1812-3_25.pdf',TIMESTAMPTZ '2026-09-29 00:00:00+00',true),
 (1009,'Фонд капитального ремонта многоквартирных домов города Москвы','REGIONAL_OPERATOR',NULL,NULL,'','https://fond.mos.ru','Официальные документы Правительства Москвы','https://www.mos.ru/upload/documents/oiv/ad_kr_605.pdf',TIMESTAMPTZ '2026-09-29 00:00:00+00',true)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,type_code=EXCLUDED.type_code,source=EXCLUDED.source,source_url=EXCLUDED.source_url,source_snapshot_at=EXCLUDED.source_snapshot_at;

DO $$
BEGIN
 IF to_regclass('gar_search_addresses') IS NOT NULL THEN
  INSERT INTO houses(address,gar_object_id,object_guid)
  SELECT full_address,object_id,object_guid FROM gar_search_addresses
  WHERE object_kind='house' AND full_address ILIKE '%ул. Арбат,%'
  ON CONFLICT(address) DO UPDATE SET gar_object_id=EXCLUDED.gar_object_id,object_guid=EXCLUDED.object_guid;
 END IF;
END $$;

INSERT INTO house_responsibility_rules(house_id,organization_id,category,place,urgency,organization_role,valid_from,source,source_url,is_demo)
SELECT h.id,o.id,v.category,v.place,'',v.role,TIMESTAMPTZ '2026-09-29 00:00:00+00','Демонстрационное правило на основе публичных данных Арбата',v.source_url,true
FROM houses h
CROSS JOIN (VALUES
 (1007::bigint,'PIPE_LEAK','COMMON_PROPERTY','PRIMARY','https://dom.gosuslugi.ru/'),
 (1007::bigint,'ELEVATOR','COMMON_PROPERTY','PRIMARY','https://dom.gosuslugi.ru/'),
 (1002::bigint,'WATER_SUPPLY','','PRIMARY','https://dom.gosuslugi.ru/'),
 (1004::bigint,'HEATING','','PRIMARY','https://dom.gosuslugi.ru/'),
 (1008::bigint,'WASTE','','PRIMARY','https://www.mos.ru/upload/documents/files/9355/ProtokolDPR-P-1812-3_25.pdf'),
 (1009::bigint,'CAPITAL_REPAIR','','PRIMARY','https://www.mos.ru/upload/documents/oiv/ad_kr_605.pdf'),
 (1005::bigint,'OUTDOOR_LIGHTING','CITY_TERRITORY','PRIMARY','https://arbat.mos.ru'),
 (1006::bigint,'OVERDUE','','ESCALATION','https://www.mos.ru/mgi/')
) AS v(org_id,category,place,role,source_url)
JOIN organizations o ON o.id=v.org_id
WHERE h.address ILIKE '%Арбат%'
AND NOT EXISTS (
 SELECT 1 FROM house_responsibility_rules r
 WHERE r.house_id=h.id AND r.organization_id=o.id AND r.category=v.category AND r.place=v.place AND r.organization_role=v.role AND r.is_demo
);
