ALTER TABLE houses ADD COLUMN IF NOT EXISTS gar_object_id bigint;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS object_guid uuid;
CREATE UNIQUE INDEX IF NOT EXISTS ux_houses_gar_object_id ON houses(gar_object_id) WHERE gar_object_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_houses_object_guid ON houses(object_guid) WHERE object_guid IS NOT NULL;

ALTER TABLE requests ADD COLUMN IF NOT EXISTS address_snapshot text NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN IF NOT EXISTS house_object_id bigint;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS house_object_guid uuid;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS apartment_object_id bigint;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS apartment_object_guid uuid;

UPDATE requests r
SET address_snapshot = h.address,
    house_object_id = COALESCE(r.house_object_id, h.gar_object_id),
    house_object_guid = COALESCE(r.house_object_guid, h.object_guid)
FROM houses h
WHERE h.id = r.house_id AND r.address_snapshot = '';
