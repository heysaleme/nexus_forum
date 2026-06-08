#!/usr/bin/env bash
# Nexus Forum backup — SQLite, Postgres, MinIO/local uploads
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
DB_TYPE="${DB_TYPE:-sqlite}"
SQLITE_DB="${SQLITE_DB:-$ROOT_DIR/backend/nexus_forum.db}"
DATABASE_URL="${DATABASE_URL:-}"
LOCAL_UPLOAD_DIR="${LOCAL_UPLOAD_DIR:-$ROOT_DIR/backend/uploads}"
MINIO_ALIAS="${MINIO_ALIAS:-local}"
MINIO_BUCKET="${MINIO_BUCKET:-nexus-forum}"

TS="$(date +%Y%m%d-%H%M%S)"
RUN_DIR="$BACKUP_DIR/$TS"

mkdir -p "$RUN_DIR"

echo "[backup] writing to $RUN_DIR"

if [[ "$DB_TYPE" == "sqlite" ]]; then
  if [[ ! -f "$SQLITE_DB" ]]; then
    echo "[backup] sqlite file not found: $SQLITE_DB" >&2
    exit 1
  fi
  DEST="$RUN_DIR/nexus_forum-$TS.db"
  cp "$SQLITE_DB" "$DEST"
  echo "[backup] sqlite -> $DEST"
elif [[ "$DB_TYPE" == "postgres" ]]; then
  if [[ -z "$DATABASE_URL" ]]; then
    echo "[backup] DATABASE_URL required for postgres" >&2
    exit 1
  fi
  DEST="$RUN_DIR/nexus_forum-$TS.sql.gz"
  pg_dump "$DATABASE_URL" | gzip > "$DEST"
  echo "[backup] postgres -> $DEST"
else
  echo "[backup] unsupported DB_TYPE=$DB_TYPE" >&2
  exit 1
fi

if [[ -d "$LOCAL_UPLOAD_DIR" ]] && [[ -n "$(ls -A "$LOCAL_UPLOAD_DIR" 2>/dev/null || true)" ]]; then
  UP_DEST="$RUN_DIR/uploads-$TS.tar.gz"
  tar -czf "$UP_DEST" -C "$(dirname "$LOCAL_UPLOAD_DIR")" "$(basename "$LOCAL_UPLOAD_DIR")"
  echo "[backup] local uploads -> $UP_DEST"
fi

if command -v mc >/dev/null 2>&1 && mc alias list "$MINIO_ALIAS" >/dev/null 2>&1; then
  MINIO_DEST="$RUN_DIR/minio-$MINIO_BUCKET-$TS"
  mkdir -p "$MINIO_DEST"
  mc mirror --overwrite=false "$MINIO_ALIAS/$MINIO_BUCKET" "$MINIO_DEST"
  echo "[backup] minio mirror -> $MINIO_DEST"
else
  echo "[backup] minio mirror skipped (mc not configured)"
fi

# Retention — never delete the backup we just created
find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -mtime +"$RETENTION_DAYS" -print -exec rm -rf {} +

echo "[backup] complete: $RUN_DIR"
