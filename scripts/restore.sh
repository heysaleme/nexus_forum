#!/usr/bin/env bash
# Nexus Forum restore from scripts/backup.sh output
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <backup-timestamp-or-path>" >&2
  echo "Example: $0 20260608-215500" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backups}"
TARGET="${1}"
DB_TYPE="${DB_TYPE:-sqlite}"
SQLITE_DB="${SQLITE_DB:-$ROOT_DIR/backend/nexus_forum.db}"
DATABASE_URL="${DATABASE_URL:-}"
LOCAL_UPLOAD_DIR="${LOCAL_UPLOAD_DIR:-$ROOT_DIR/backend/uploads}"
MINIO_ALIAS="${MINIO_ALIAS:-local}"
MINIO_BUCKET="${MINIO_BUCKET:-nexus-forum}"

if [[ -d "$TARGET" ]]; then
  RUN_DIR="$TARGET"
elif [[ -d "$BACKUP_DIR/$TARGET" ]]; then
  RUN_DIR="$BACKUP_DIR/$TARGET"
else
  echo "[restore] backup not found: $TARGET" >&2
  exit 1
fi

echo "[restore] from $RUN_DIR"

if [[ "$DB_TYPE" == "sqlite" ]]; then
  SRC="$(find "$RUN_DIR" -name 'nexus_forum-*.db' | head -1)"
  if [[ -z "$SRC" ]]; then
    echo "[restore] sqlite backup missing" >&2
    exit 1
  fi
  if [[ -f "$SQLITE_DB" ]]; then
    cp "$SQLITE_DB" "$SQLITE_DB.pre-restore-$(date +%Y%m%d-%H%M%S)"
  fi
  cp "$SRC" "$SQLITE_DB"
  chmod 644 "$SQLITE_DB" 2>/dev/null || true
  echo "[restore] sqlite restored -> $SQLITE_DB"
elif [[ "$DB_TYPE" == "postgres" ]]; then
  SRC="$(find "$RUN_DIR" -name 'nexus_forum-*.sql.gz' | head -1)"
  if [[ -z "$SRC" || -z "$DATABASE_URL" ]]; then
    echo "[restore] postgres backup or DATABASE_URL missing" >&2
    exit 1
  fi
  gunzip -c "$SRC" | psql "$DATABASE_URL"
  echo "[restore] postgres restored"
fi

UP_SRC="$(find "$RUN_DIR" -name 'uploads-*.tar.gz' | head -1)"
if [[ -n "$UP_SRC" ]]; then
  mkdir -p "$(dirname "$LOCAL_UPLOAD_DIR")"
  tar -xzf "$UP_SRC" -C "$(dirname "$LOCAL_UPLOAD_DIR")"
  echo "[restore] uploads restored -> $LOCAL_UPLOAD_DIR"
fi

MINIO_SRC="$(find "$RUN_DIR" -maxdepth 1 -type d -name 'minio-*' | head -1)"
if [[ -n "$MINIO_SRC" ]] && command -v mc >/dev/null 2>&1 && mc alias list "$MINIO_ALIAS" >/dev/null 2>&1; then
  mc mirror --overwrite=false "$MINIO_SRC" "$MINIO_ALIAS/$MINIO_BUCKET"
  echo "[restore] minio restored"
fi

echo "[restore] complete"
