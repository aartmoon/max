#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 FIXTURE_DIR" >&2
  exit 2
fi

fixture_dir=$1
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file=${COMPOSE_FILE:-docker-compose.yml}
case "$fixture_dir" in
  /*) ;;
  *) fixture_dir="$PWD/$fixture_dir" ;;
esac
test -f "$fixture_dir/manifest.sha256" || { echo "missing manifest.sha256" >&2; exit 2; }

stage_dir=$(mktemp -d "${TMPDIR:-/tmp}/arbat-verify.XXXXXX")
container_dir="/tmp/arbat-verify-$$"
cleanup() {
  docker compose -f "$compose_file" exec -T postgres rm -rf "$container_dir" >/dev/null 2>&1 || true
  rm -rf "$stage_dir"
}
trap cleanup EXIT HUP INT TERM
docker compose -f "$compose_file" exec -T postgres sh -c 'mkdir -p "$1" && chmod 0777 "$1"' sh "$container_dir"

set --
for key in address_objects addr_object_params addr_object_divisions adm_hierarchy mun_hierarchy houses house_params apartments apartment_params carplaces carplace_params rooms room_params steads stead_params change_history normative_docs reestr_objects; do
  set -- "$@" -v "${key}_file=${container_dir}/${key}.csv"
done
docker compose -f "$compose_file" exec -T -e "TARGET_DB=${TARGET_DB:-}" postgres sh -c \
  'db=${TARGET_DB:-$POSTGRES_DB}; psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$db" "$@"' sh "$@" \
  < "$root/scripts/export-demo-database.sql"
docker compose -f "$compose_file" cp "postgres:${container_dir}/." "$stage_dir/"

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

: > "$stage_dir/manifest.sha256"
for file in "$stage_dir"/*.csv; do
  base=$(basename -- "$file")
  printf '%s  %s\n' "$(hash_file "$file")" "$base" >> "$stage_dir/manifest.sha256"
done

if ! cmp -s "$fixture_dir/manifest.sha256" "$stage_dir/manifest.sha256"; then
  echo "fixture differs from database" >&2
  diff -u "$fixture_dir/manifest.sha256" "$stage_dir/manifest.sha256" >&2 || true
  exit 1
fi
echo "fixture parity: ok"
