#!/bin/sh
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 OUTPUT_DIR HOUSE_ID_1,...,HOUSE_ID_10" >&2
  exit 2
fi

output_dir=$1
house_ids=$2
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file=${COMPOSE_FILE:-docker-compose.yml}
case "$output_dir" in
  /*) ;;
  *) output_dir="$PWD/$output_dir" ;;
esac

if ! printf '%s\n' "$house_ids" | grep -Eq '^[1-9][0-9]*(,[1-9][0-9]*){9}$'; then
  echo "HOUSE_IDS must contain exactly ten comma-separated positive integers" >&2
  exit 2
fi

parent_dir=$(dirname -- "$output_dir")
mkdir -p "$parent_dir"
stage_dir=$(mktemp -d "$parent_dir/.arbat-fixture.XXXXXX")
container_dir="/tmp/arbat-fixture-$$"
cleanup() {
  docker compose -f "$compose_file" exec -T postgres rm -rf "$container_dir" >/dev/null 2>&1 || true
  if [ -d "$stage_dir" ]; then rm -rf "$stage_dir"; fi
}
trap cleanup EXIT HUP INT TERM

docker compose -f "$compose_file" exec -T postgres sh -c 'mkdir -p "$1" && chmod 0777 "$1"' sh "$container_dir"

set --
for key in address_objects addr_object_params addr_object_divisions adm_hierarchy mun_hierarchy houses house_params apartments apartment_params carplaces carplace_params rooms room_params steads stead_params change_history normative_docs reestr_objects; do
  set -- "$@" -v "${key}_file=${container_dir}/${key}.csv"
done
set -- "$@" -v "source_meta_file=${container_dir}/source_meta.csv" -v "house_ids=$house_ids"

docker compose -f "$compose_file" exec -T -e "TARGET_DB=${TARGET_DB:-}" postgres sh -c \
  'db=${TARGET_DB:-$POSTGRES_DB}; psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$db" "$@"' sh "$@" \
  < "$root/scripts/export-arbat-fixture.sql"

docker compose -f "$compose_file" cp "postgres:${container_dir}/." "$stage_dir/"

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

: > "$stage_dir/manifest.sha256"
: > "$stage_dir/manifest.rows"
for file in "$stage_dir"/*.csv; do
  base=$(basename -- "$file")
  [ "$base" = source_meta.csv ] && continue
  rows=$(awk 'END { if (NR > 0) print NR - 1; else print 0 }' "$file")
  digest=$(hash_file "$file")
  printf '%s  %s\n' "$digest" "$base" >> "$stage_dir/manifest.sha256"
  printf '%s  %s\n' "$rows" "$base" >> "$stage_dir/manifest.rows"
done

meta=$(sed -n '2p' "$stage_dir/source_meta.csv")
database=$(printf '%s\n' "$meta" | cut -d, -f1)
source_date=$(printf '%s\n' "$meta" | cut -d, -f2)
street_id=$(printf '%s\n' "$meta" | cut -d, -f3)
selected_houses=$(printf '%s\n' "$meta" | cut -d, -f4 | tr '|' ',')
generated_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
cat > "$stage_dir/source.json" <<EOF
{
  "database": "$database",
  "sourceDate": "$source_date",
  "generatedAt": "$generated_at",
  "streetId": $street_id,
  "houseIds": [$selected_houses]
}
EOF
rm "$stage_dir/source_meta.csv"

if [ -e "$output_dir" ]; then
  echo "output already exists: $output_dir" >&2
  exit 3
fi
mv "$stage_dir" "$output_dir"
trap - EXIT HUP INT TERM
docker compose -f "$compose_file" exec -T postgres rm -rf "$container_dir" >/dev/null
echo "fixture exported to $output_dir"
