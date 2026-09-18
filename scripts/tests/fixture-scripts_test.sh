#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)

require_file() {
  test -f "$root/$1" || { echo "missing $1" >&2; exit 1; }
}

require_text() {
  file=$1
  pattern=$2
  grep -Eq "$pattern" "$root/$file" || {
    echo "$file does not match: $pattern" >&2
    exit 1
  }
}

require_file scripts/export-arbat-fixture.sql
require_file scripts/export-demo-database.sql
require_file scripts/export-arbat-fixture.sh
require_file scripts/verify-arbat-fixture.sh
require_file scripts/reset-production-demo.sh

require_text scripts/export-arbat-fixture.sql 'BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY'
require_text scripts/export-arbat-fixture.sql 'Москв'
require_text scripts/export-arbat-fixture.sql 'Арбат'
require_text scripts/export-arbat-fixture.sql 'count\(\*\).*10'
require_text scripts/export-arbat-fixture.sql 'gar_address_objects'
require_text scripts/export-arbat-fixture.sql 'gar_addr_object_params'
require_text scripts/export-arbat-fixture.sql 'gar_addr_object_divisions'
require_text scripts/export-arbat-fixture.sql 'gar_adm_hierarchy'
require_text scripts/export-arbat-fixture.sql 'gar_mun_hierarchy'
require_text scripts/export-arbat-fixture.sql 'gar_houses'
require_text scripts/export-arbat-fixture.sql 'gar_house_params'
require_text scripts/export-arbat-fixture.sql 'gar_apartments'
require_text scripts/export-arbat-fixture.sql 'gar_apartment_params'
require_text scripts/export-arbat-fixture.sql 'gar_carplaces'
require_text scripts/export-arbat-fixture.sql 'gar_carplace_params'
require_text scripts/export-arbat-fixture.sql 'gar_rooms'
require_text scripts/export-arbat-fixture.sql 'gar_room_params'
require_text scripts/export-arbat-fixture.sql 'gar_steads'
require_text scripts/export-arbat-fixture.sql 'gar_stead_params'
require_text scripts/export-arbat-fixture.sql 'gar_change_history'
require_text scripts/export-arbat-fixture.sql 'gar_normative_docs'
require_text scripts/export-arbat-fixture.sql 'gar_reestr_objects'
require_text scripts/export-arbat-fixture.sql '^COMMIT;'

require_text scripts/export-arbat-fixture.sh 'ON_ERROR_STOP=1'
require_text scripts/export-arbat-fixture.sh 'sha256sum|shasum -a 256'
require_text scripts/export-arbat-fixture.sh 'source.json'
require_text scripts/verify-arbat-fixture.sh 'manifest.sha256'
require_text scripts/reset-production-demo.sh 'EXPECTED_DATABASE'
require_text scripts/reset-production-demo.sh 'current_database\(\)'
require_text scripts/reset-production-demo.sh 'stop.*frontend.*backend.*gar-init|stop.*backend.*gar-init.*frontend'
require_text scripts/reset-production-demo.sh 'export-arbat-fixture.sh'
require_text scripts/reset-production-demo.sh 'manifest.sha256'
require_text scripts/reset-production-demo.sh 'pg_dump.*-Fc'
require_text scripts/reset-production-demo.sh 'pg_restore.*--list'
require_text scripts/reset-production-demo.sh 'RESET.*expected_database'
require_text scripts/reset-production-demo.sh 'DROP SCHEMA public CASCADE'
require_text scripts/reset-production-demo.sh 'ON_ERROR_STOP=1'
require_text docker-compose.yml 'GAR_MODE:[[:space:]]*demo'
require_text docker-compose.prod.yml 'GAR_MODE:[[:space:]]*demo'
require_text docker-compose.prod.yml 'condition:[[:space:]]*service_completed_successfully'
require_text deploy.sh 'pull postgres gar-init backend frontend'
require_text deploy.sh '\.demo-reset-maintenance'
require_text deploy.sh 'stop frontend backend gar-init'
if grep -Eq 'GAR_ALLOW_DEMO_DATA' "$root/docker-compose.yml" "$root/docker-compose.prod.yml"; then
  echo "Compose still exposes GAR_ALLOW_DEMO_DATA" >&2
  exit 1
fi
if grep -A3 -E '^[[:space:]]*gar-init:' "$root/docker-compose.prod.yml" | grep -Eq 'profiles:'; then
  echo "production gar-init is still hidden behind a profile" >&2
  exit 1
fi

if grep -Eiq 'postgres://|password|POSTGRES_PASSWORD' "$root/gar-init/internal/db/fixtures/source.json" 2>/dev/null; then
  echo "fixture source manifest contains a credential-like value" >&2
  exit 1
fi

echo "fixture script contracts: ok"
