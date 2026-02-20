#!/bin/sh
# Compare goprecords (Go) output with guprecords (Raku) on the same stats dir.
# Usage: ./scripts/compare-with-raku.sh [stats-dir]
# Default stats-dir: ../uprecords/stats (relative to repo root) or set UPRECORDS_STATS.

set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
STATS="${1:-${UPRECORDS_STATS:-$REPO_ROOT/../uprecords/stats}}"
RAKU_REPO="${GUPRECORDS_RAKU:-$REPO_ROOT/../guprecords}"

if [ ! -d "$STATS" ]; then
  echo "Stats dir not found: $STATS" >&2
  echo "Usage: $0 [stats-dir]" >&2
  echo "  or set UPRECORDS_STATS" >&2
  exit 1
fi
if [ ! -f "$RAKU_REPO/guprecords.raku" ]; then
  echo "Raku guprecords not found: $RAKU_REPO/guprecords.raku" >&2
  echo "Set GUPRECORDS_RAKU to the guprecords (Raku) repo." >&2
  exit 1
fi

GO_BIN="$REPO_ROOT/goprecords"
if [ ! -x "$GO_BIN" ]; then
  echo "Build goprecords first (e.g. mage build)" >&2
  exit 1
fi

mkdir -p /tmp/goprecords-compare
R=/tmp/goprecords-compare/raku
G=/tmp/goprecords-compare/go

echo "Stats dir: $STATS"
echo "Raku repo: $RAKU_REPO"
echo ""

# Single report: Host Uptime
echo "=== Host Uptime (limit 10) ==="
raku "$RAKU_REPO/guprecords.raku" --stats-dir="$STATS" --category=Host --metric=Uptime --limit=10 --output-format=Plaintext 2>/dev/null > "$R.host_uptime.txt"
"$GO_BIN" -stats-dir="$STATS" -category=Host -metric=Uptime -limit=10 -output-format=Plaintext 2>/dev/null > "$G.host_uptime.txt"
if diff -q "$R.host_uptime.txt" "$G.host_uptime.txt" >/dev/null; then
  echo "OK (identical)"
else
  diff -u "$R.host_uptime.txt" "$G.host_uptime.txt" || true
fi

# Single report: Host Boots, Markdown
echo ""
echo "=== Host Boots Markdown (limit 5) ==="
raku "$RAKU_REPO/guprecords.raku" --stats-dir="$STATS" --category=Host --metric=Boots --limit=5 --output-format=Markdown 2>/dev/null > "$R.host_boots_md.txt"
"$GO_BIN" -stats-dir="$STATS" -category=Host -metric=Boots -limit=5 -output-format=Markdown 2>/dev/null > "$G.host_boots_md.txt"
if diff -q "$R.host_boots_md.txt" "$G.host_boots_md.txt" >/dev/null; then
  echo "OK (identical)"
else
  diff -u "$R.host_boots_md.txt" "$G.host_boots_md.txt" || true
fi

# --all (known: description word-wrap may differ in one section)
echo ""
echo "=== --all limit 5 ==="
raku "$RAKU_REPO/guprecords.raku" --stats-dir="$STATS" --all --limit=5 --output-format=Plaintext 2>/dev/null > "$R.all.txt"
"$GO_BIN" -stats-dir="$STATS" -all -limit=5 -output-format=Plaintext 2>/dev/null > "$G.all.txt"
if diff -q "$R.all.txt" "$G.all.txt" >/dev/null; then
  echo "OK (identical)"
else
  echo "Differences (often only description word-wrap):"
  diff -u "$R.all.txt" "$G.all.txt" | head -40 || true
fi

echo ""
echo "Done. Raku output: $R.*.txt  Go output: $G.*.txt"
