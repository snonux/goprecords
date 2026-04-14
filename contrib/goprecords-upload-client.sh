#!/bin/sh
set -e
GOPRECORDS_BASE_URL="${GOPRECORDS_BASE_URL:-https://goprecords.f3s.buetow.org}"
GOPRECORDS_HOST="${GOPRECORDS_HOST:?set GOPRECORDS_HOST (e.g. f0, pi0)}"
GOPRECORDS_TOKEN_FILE="${GOPRECORDS_TOKEN_FILE:-/etc/goprecords-upload.token}"
PATH="/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:${PATH}"

if ! test -r "$GOPRECORDS_TOKEN_FILE"; then
	echo "goprecords-upload-client: cannot read $GOPRECORDS_TOKEN_FILE" >&2
	exit 1
fi
TOKEN=$(tr -d '\n\r' <"$GOPRECORDS_TOKEN_FILE")

upload() {
	kind=$1
	file=$2
	if ! test -f "$file"; then
		echo "goprecords-upload-client: skip $kind (no $file)" >&2
		return 0
	fi
	curl -fsS -X PUT --data-binary "@${file}" \
		-H "Authorization: Bearer ${TOKEN}" \
		"${GOPRECORDS_BASE_URL}/upload/${GOPRECORDS_HOST}/${kind}"
}

records_path=
if test -f /var/spool/uptimed/records; then
	records_path=/var/spool/uptimed/records
elif test -f /var/db/uptimed/records; then
	records_path=/var/db/uptimed/records
elif test -f /usr/local/var/uptimed/records; then
	records_path=/usr/local/var/uptimed/records
else
	echo "goprecords-upload-client: no uptimed records file found" >&2
	exit 1
fi

tmp=$(mktemp)
trap 'rm -f "$tmp"' 0 INT TERM HUP

upload records "$records_path"

if command -v uprecords >/dev/null 2>&1; then
	uprecords -a -m 100 >"$tmp"
	upload txt "$tmp"
	uprecords -a | grep '^->' >"$tmp" || true
	if test -s "$tmp"; then
		upload cur.txt "$tmp"
	fi
fi

if test -r /etc/os-release; then
	upload os.txt /etc/os-release
else
	uname -a >"$tmp"
	upload os.txt "$tmp"
fi

if test -r /proc/cpuinfo; then
	upload cpuinfo.txt /proc/cpuinfo
elif test -r /var/run/dmesg.boot; then
	upload cpuinfo.txt /var/run/dmesg.boot
else
	sysctl hw.model hw.ncpu hw.machine >"$tmp" 2>/dev/null || uname -a >"$tmp"
	upload cpuinfo.txt "$tmp"
fi
