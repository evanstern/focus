#!/usr/bin/env bash
# Handler for the coda plugin system.
# Sourced by coda's plugin dispatcher; functions below are called with subcommand args.

_coda_focus_require() {
    if ! command -v focus &>/dev/null; then
        echo "error: focus not found on PATH" >&2
        echo "Install: make install (from the focus repo)" >&2
        return 1
    fi
}

_coda_focus() {
    _coda_focus_require || return 1
    focus "$@"
}

_coda_focus_new() {
    _coda_focus_require || return 1
    local title="" project=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --title)   title="$2"; shift 2 ;;
            --project) project="$2"; shift 2 ;;
            *)         shift ;;
        esac
    done
    if [[ -z "$title" ]]; then
        echo "error: --title is required" >&2
        return 1
    fi
    local args=("new" "$title")
    [[ -n "$project" ]] && args+=("$project")
    focus "${args[@]}"
}

_coda_focus_transition() {
    _coda_focus_require || return 1
    local cmd="$1" ref="" force=false
    shift
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --ref)   ref="$2"; shift 2 ;;
            --force) force=true; shift ;;
            *)       shift ;;
        esac
    done
    if [[ -z "$ref" ]]; then
        echo "error: --ref is required" >&2
        return 1
    fi
    local args=("$cmd" "$ref")
    $force && args+=("--force")
    focus "${args[@]}"
}

_coda_focus_show()     { _coda_focus_transition show "$@"; }
_coda_focus_activate() { _coda_focus_transition activate "$@"; }
_coda_focus_done()     { _coda_focus_transition done "$@"; }
_coda_focus_park()     { _coda_focus_transition park "$@"; }
_coda_focus_kill()     { _coda_focus_transition kill "$@"; }
_coda_focus_edit()     { _coda_focus_transition edit "$@"; }

_coda_focus_list() {
    _coda_focus_require || return 1
    local status="" project=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --status)  status="$2"; shift 2 ;;
            --project) project="$2"; shift 2 ;;
            *)         shift ;;
        esac
    done
    local args=("list")
    [[ -n "$status" ]] && args+=("$status")
    [[ -n "$project" ]] && args+=("--project" "$project")
    args+=("--no-color")
    focus "${args[@]}"
}

_coda_focus_intent() {
    _coda_focus_require || return 1
    local message=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --message) message="$2"; shift 2 ;;
            *)         shift ;;
        esac
    done
    if [[ -n "$message" ]]; then
        focus intent "$message"
    else
        focus intent
    fi
}
