INSERT INTO house_responsibility_rules(
 house_id,organization_id,category,place,urgency,organization_role,
 valid_from,source,source_url,is_demo
)
SELECT h.id,1007,'*',v.place,'','PRIMARY',
 TIMESTAMPTZ '2026-09-30 00:00:00+00',
 'Резервное правило управляющей организации для дома',
 'https://www.mos.ru/',true
FROM houses h
CROSS JOIN (VALUES ('COMMON_PROPERTY'),('APARTMENT'),('YARD')) AS v(place)
WHERE h.address ILIKE '%Арбат%'
AND EXISTS (SELECT 1 FROM organizations o WHERE o.id=1007 AND o.active)
AND NOT EXISTS (
 SELECT 1 FROM house_responsibility_rules r
 WHERE r.house_id=h.id AND r.category='*' AND r.place=v.place
  AND r.organization_role='PRIMARY' AND r.active
);
