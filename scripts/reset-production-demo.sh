#!/bin/sh
set -eu

if [ "$#" -ne 3 ]; then
  echo "usage: $0 EXPECTED_DATABASE BACKUP_DIR FIXTURE_DIR" >&2
  exit 2
fi

expected_database=$1
backup_dir=$2
fixture_dir=$3
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file=${COMPOSE_FILE:-docker-compose.prod.yml}
cd "$root"

case "$expected_database" in
  *[!A-Za-z0-9_.-]*|'') echo "invalid EXPECTED_DATABASE" >&2; exit 2 ;;
esac
case "$backup_dir" in /*) ;; *) backup_dir="$PWD/$backup_dir" ;; esac
case "$fixture_dir" in /*) ;; *) fixture_dir="$PWD/$fixture_dir" ;; esac

test -f "$fixture_dir/manifest.sha256" || { echo "missing fixture manifest.sha256" >&2; exit 2; }
test -f "$fixture_dir/manifest.rows" || { echo "missing fixture manifest.rows" >&2; exit 2; }
test -f "$fixture_dir/source.json" || { echo "missing fixture source.json" >&2; exit 2; }

house_ids=$(sed -n 's/.*"houseIds": \[\(.*\)\].*/\1/p' "$fixture_dir/source.json" | tr -d ' ')
if ! printf '%s\n' "$house_ids" | grep -Eq '^[1-9][0-9]*(,[1-9][0-9]*){9}$'; then
  echo "fixture source.json must contain exactly ten house IDs" >&2
  exit 2
fi

live_database=$(docker compose -f "$compose_file" exec -T -e "TARGET_DB=$expected_database" postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$TARGET_DB" -Atqc "SELECT current_database()"')
if [ "$live_database" != "$expected_database" ]; then
  echo "database mismatch: expected $expected_database, connected to $live_database" >&2
  exit 3
fi

echo "Stopping application services before the final parity snapshot"
docker compose -f "$compose_file" stop frontend backend gar-init

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/production-demo-reset.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM
live_fixture="$work_dir/live-fixture"
COMPOSE_FILE="$compose_file" TARGET_DB="$expected_database" \
  sh scripts/export-arbat-fixture.sh "$live_fixture" "$house_ids"

if ! cmp -s "$fixture_dir/manifest.sha256" "$live_fixture/manifest.sha256" || \
   ! cmp -s "$fixture_dir/manifest.rows" "$live_fixture/manifest.rows"; then
  echo "production rows differ from the committed fixture; reset aborted" >&2
  diff -u "$fixture_dir/manifest.sha256" "$live_fixture/manifest.sha256" >&2 || true
  diff -u "$fixture_dir/manifest.rows" "$live_fixture/manifest.rows" >&2 || true
  exit 4
fi

mkdir -p "$backup_dir"
umask 077
timestamp=$(date -u '+%Y%m%dT%H%M%SZ')
backup_file="$backup_dir/${expected_database}-${timestamp}.dump"
partial_backup="$backup_file.partial"
echo "Creating full compressed backup at $backup_file"
docker compose -f "$compose_file" exec -T -e "TARGET_DB=$expected_database" postgres sh -c \
  'pg_dump -Fc -U "$POSTGRES_USER" -d "$TARGET_DB"' > "$partial_backup"
test -s "$partial_backup" || { echo "backup is empty" >&2; exit 5; }
docker compose -f "$compose_file" exec -T postgres pg_restore --list < "$partial_backup" >/dev/null
mv "$partial_backup" "$backup_file"

fixture_digest=$(if command -v sha256sum >/dev/null 2>&1; then sha256sum "$fixture_dir/manifest.sha256"; else shasum -a 256 "$fixture_dir/manifest.sha256"; fi | awk '{print $1}')
echo "Target database: $expected_database"
echo "Fixture manifest SHA-256: $fixture_digest"
echo "Verified backup: $backup_file"
printf 'Type RESET %s to continue: ' "$expected_database"
IFS= read -r confirmation
if [ "$confirmation" != "RESET $expected_database" ]; then
  echo "confirmation did not match; reset aborted" >&2
  exit 6
fi

docker compose -f "$compose_file" exec -T -e "TARGET_DB=$expected_database" postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$TARGET_DB"' <<'SQL'
DROP SCHEMA public CASCADE;
CREATE SCHEMA public AUTHORIZATION CURRENT_USER;
GRANT ALL ON SCHEMA public TO CURRENT_USER;
GRANT USAGE ON SCHEMA public TO PUBLIC;
SQL

docker compose -f "$compose_file" up --pull always gar-init
docker compose -f "$compose_file" up -d --pull always backend frontend
docker compose -f "$compose_file" exec -T backend wget -q -O - http://127.0.0.1:8080/api/health

echo "Production database reset completed; backup retained at $backup_file"
