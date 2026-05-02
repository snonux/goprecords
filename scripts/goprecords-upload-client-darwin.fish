#!/usr/bin/env fish

if test -z "$GOPRECORDS_HOST"
    echo "goprecords-upload-client-darwin: GOPRECORDS_HOST is not set" >&2
    exit 1
end

set -g _gpr_base_url $GOPRECORDS_BASE_URL
if test -z "$_gpr_base_url"
    set -g _gpr_base_url "https://goprecords.f3s.buetow.org"
end

set -l config_dir $XDG_CONFIG_HOME
if test -z "$config_dir"
    set config_dir "$HOME/.config"
end
set -g _gpr_token_file $GOPRECORDS_TOKEN_FILE
if test -z "$_gpr_token_file"
    set -g _gpr_token_file "$config_dir/goprecords-upload-$GOPRECORDS_HOST/token"
end

if not test -r "$_gpr_token_file"
    echo "goprecords-upload-client-darwin: cannot read $_gpr_token_file" >&2
    exit 1
end

set -g _gpr_token (string trim (cat "$_gpr_token_file"))

function _upload
    set -l kind $argv[1]
    set -l file $argv[2]
    if not test -f "$file"
        echo "goprecords-upload-client-darwin: skip $kind (no $file)" >&2
        return 0
    end
    curl -fsS -X PUT --data-binary "@$file" \
        -H "Authorization: Bearer $_gpr_token" \
        "$_gpr_base_url/upload/$GOPRECORDS_HOST/$kind"
end

function _upload_required
    set -l kind $argv[1]
    set -l file $argv[2]
    _upload "$kind" "$file"; or exit 1
end

function _find_records
    if test -n "$GOPRECORDS_RECORDS_FILE"
        if test -f "$GOPRECORDS_RECORDS_FILE"
            echo "$GOPRECORDS_RECORDS_FILE"
            return 0
        end
        echo "goprecords-upload-client-darwin: no uptimed records file at GOPRECORDS_RECORDS_FILE=$GOPRECORDS_RECORDS_FILE" >&2
        exit 1
    end
    for p in /usr/local/var/uptimed/records /opt/homebrew/var/uptimed/records
        if test -f "$p"
            echo $p
            return 0
        end
    end
    echo "goprecords-upload-client-darwin: no uptimed records file found" >&2
    exit 1
end

set -l records_path (_find_records)
set -g _gpr_tmp (mktemp /tmp/goprecords-darwin.XXXXXX)

function _cleanup_tmp --on-event fish_exit
    if test -n "$_gpr_tmp"
        rm -f "$_gpr_tmp"
    end
end

_upload_required records "$records_path"

if command -v uprecords >/dev/null 2>&1
    uprecords -a -m 100 >"$_gpr_tmp"
    _upload_required txt "$_gpr_tmp"

    uprecords -a | grep '^->' >"$_gpr_tmp" 2>/dev/null; or true
    if test -s "$_gpr_tmp"
        _upload_required cur.txt "$_gpr_tmp"
    end
end

if command -v sw_vers >/dev/null 2>&1
    sw_vers >"$_gpr_tmp"
else
    uname -a >"$_gpr_tmp"
end
_upload_required os.txt "$_gpr_tmp"

sysctl hw.model hw.ncpu hw.machine >"$_gpr_tmp" 2>/dev/null; or uname -a >"$_gpr_tmp"
_upload_required cpuinfo.txt "$_gpr_tmp"
