#!/usr/bin/bash

log () { "$DEBUG" && echo -e "[$(date '+%T.%3N')] $*" >> ~/mybash.log; }
errmsg () { echo "$@" 1>&2; }

declare -g bctils_err=""

BCTILS_COMPILE_DIR="${BCTILS_COMPILE_DIR:-"$HOME/.config/bctils"}"
BCTILS_BIN_PATH="$(which bctils)"
BCTILS_SH_PATH="$(realpath "${BASH_SOURCE[0]}")"
DEBUG=false

bctils_cli_register () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"

  log "registering: $cli_name"
  unset "__bctils_data_args_${cli_name_clean}"
  declare -g -a "__bctils_data_args_${cli_name_clean}"
  unset "__bctils_data_errors_${cli_name_clean}"
  declare -g -a "__bctils_data_errors_${cli_name_clean}"
}

# $1: cli_name
# $2: type 'opt'|'pos'
# $3: option name if 'opt'
bctils_cli_add () {
  local cli_name="$1"; shift
  local arg_type="$1"; shift
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"
  local -n bctils_data_args="__bctils_data_args_${cli_name_clean}"
  local -n bctils_data_errors="__bctils_data_errors_${cli_name_clean}"
  local parts_rest=()
  local -a args=("$0")
  local -A options=()
  local arg_type

  # allow -p to be before "--option-name"
  if [[ "$1" =~ ^"-p=" ]]; then
    IFS="=" read -r arg arg_value <<< "$1"; shift
    part_parser=" $arg=\"$arg_value\""
  else
    part_parser=""
  fi

  case "$arg_type" in
    "cfg") part_arg_1=" \"$1\""; shift ;;
    "opt") part_arg_1=" \"$1\""; shift ;;
    "pos") part_arg_1="" ;;
    *)
      error_str="bctils_cli_add error: second argument must be type opt or pos"
      errmsg "$error_str"
      bctils_data_errors+=("$error_str")
      bctils_err="second argument must be type opt or pos"
      return 1 ;;
  esac

  bctils_err=""

  while true; do
    if [[ -z "$1" ]]; then break; fi
    IFS="=" read -r arg arg_value <<< "$1"; shift
    case "$arg" in
      "--choices"|\
      "--closure"|\
      "-p")
        parts_rest+=(" $arg=\"$arg_value\"") ;;
      "--") break ;;
      -*)
        errmsg "error: unknown option $arg ${arg_value}" ;;
      *) args+=("$arg") ;;
    esac
  done

  part_argtype="$arg_type"
  printf -v arg_str '%s' "$part_argtype" "$part_parser" "$part_arg_1" "${parts_rest[@]}"
  bctils_data_args+=("$arg_str")
}

bctils_cli_compile () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"
  bctils_err=""

  local -A options=()
  local -a args=()
  if ! TEMP=$(getopt -o '-h' --longoptions 'source' -- "$@"); then errmsg "failed to parse args"; return 1; fi
  eval set -- "$TEMP"; unset TEMP
  while true; do
    case "$1" in
      "--source")
        options["source"]=1
        shift 1
        ;;
      "--") break ;;
      *) args+=("$1"); shift ;;
    esac
  done

  out_file="${args[1]:-"$BCTILS_COMPILE_DIR/${cli_name}_complete.sh"}"

  # shellcheck disable=SC2178
  local -n bctils_data_args="__bctils_data_args_${cli_name_clean}"
  # shellcheck disable=SC2178
  local -n bctils_data_errors="__bctils_data_errors_${cli_name_clean}"

  if [[ "${#bctils_data_errors[@]}" -gt 0 ]]; then
    errmsg "bctils_cli_compile: unable to compile with errors"
    1>&2 printf "%s %s\n" '-' "${bctils_data_errors[@]}"
    # shellcheck disable=SC2034
    bctils_err="cannot compile with errors adding arguments"
    return
  fi

  mkdir -p "$(dirname "$out_file")"
  if ! printf '%s\n' "${bctils_data_args[@]}" | bctils "$cli_name" > "$out_file"
  then
    # shellcheck disable=SC2034
    bctils_err="bctils compile failed"
    return
  fi

  if [[ "${options["source"]}" == 1 ]]; then
    log "sourcing $out_file"
    # shellcheck disable=SC1090
    source "$out_file"
  fi
}

# todo: sourcing 6ms
# todo: bctils_autogen call 11ms (cached)
bctils_autogen () {
  local args_verbatim
  args_verbatim="${FUNCNAME[0]} $(printf " %q" "${@}")"
  local -a watchfiles=()
  local -A options=()
  local -a args=("$0")
  if ! TEMP=$(getopt -o '-h' --longoptions 'source,lang:,outfile:,closurepipe:,watch-file:,cliname:' -- "$@"); then errmsg "failed to parse args"; return 1; fi
  eval set -- "$TEMP"; unset TEMP
  while true; do
    case "$1" in
      "--cliname")
        options["cliname"]="$2"; shift 2 ;;
      "--watch-file")
        watchfiles+=("$2"); shift 2;;
      "--closurepipe")
        options["closurepipe"]="$2"; shift 2 ;;
      "--outfile")
        options["outfile"]="$2"; shift 2 ;;
      "--lang")
        options["lang"]="$2"; shift 2 ;;
      "--source")
        options["source"]=1; shift ;;
      "--") break ;;
      *) args+=("$1"); shift ;;
    esac
  done

  lang="${options["lang"]}"
  files=("${args[@]:1}")
  cli_name="${options[cliname]:-"$(basename "${files[0]}")"}"
  out_file="${options[outfile]:-"$BCTILS_COMPILE_DIR/${cli_name}_complete.sh"}"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"

  log "$cli_name : ${lang} autogen for files : ${files[*]}"
  reload_files=( "${files[@]}" "$BCTILS_BIN_PATH" "$BCTILS_SH_PATH" "${watchfiles[@]}")
  cache_file="$HOME/.cache/bctils_autogen_md5_nonreload_$cli_name_clean"
  reload_files_md5="$(stat  --printf '%Y' "${reload_files[@]}")"  # 2-3ms
  cache_content="$(cat "$cache_file" 2>/dev/null)"                # 3ms
  if [[ "$cache_content" != "$reload_files_md5" ]]
  then
    autogen_args=(
      -autogen-lang py
      -autogen-src "${files[0]}"
      -autogen-outfile "$out_file"
      -autogen-extra-watch "$(which bctils)"  # todo: cache this at top
      -autogen-extra-watch "$(realpath "${BASH_SOURCE[0]}")"
    )
    for watchfile in "${watchfiles[@]}"; do
      autogen_args+=(-autogen-extra-watch "$watchfile")
    done
    autogen_args+=("$cli_name" "$args_verbatim")

    if [[ -n "${options[closurepipe]}" ]]; then
      local out_dir
      out_dir="$(dirname "$out_file")"
      if [[ ! -d "$out_dir" ]]; then
        mkdir -p "$out_dir"
      fi

      if ! "${options[closurepipe]}" | bctils "${autogen_args[@]}" > "$out_file"
      then
        # shellcheck disable=SC2034
        bctils_err="bctils autogen failed"
        return 1
      fi
    else
      if ! bctils "${autogen_args[@]}" > "$out_file"
      then
        # shellcheck disable=SC2034
        bctils_err="bctils autogen failed"
        return 1
      fi
    fi
    echo "$reload_files_md5" > "$cache_file"
  fi

  if [[ "${options["source"]}" == 1 ]]; then
    log "sourcing $out_file"
    # shellcheck disable=SC1090
    source "$out_file"
  fi
}

__bctils_dump () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"

  # shellcheck disable=SC2178
  local -n bctils_data_args="__bctils_data_args_${cli_name_clean}"

  log "==== dumping start '$cli_name' ===="
  local i
  for i in "${!bctils_data_args[@]}"; do
    log "bctils_data_args[$i]: ${bctils_data_args[$i]}"
  done
}
