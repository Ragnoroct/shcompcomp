log () { echo -e "[$(date '+%T.%3N')] $*" >> ~/mybash.log; }
errmsg () { echo "$@" 1>&2; }

declare -g bctils_err=""

BCTILS_COMPILE_DIR="${BCTILS_COMPILE_DIR:-"$HOME/.config/bctils"}"

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

bctils_autogen () {
  local args_verbatim
  args_verbatim="${FUNCNAME[0]} $(printf " %q" "${@}")"

  local -A options=()
  local -a args=("$0")
  if ! TEMP=$(getopt -o '-h' --longoptions 'source,lang:,outfile:' -- "$@"); then errmsg "failed to parse args"; return 1; fi
  eval set -- "$TEMP"; unset TEMP
  while true; do
    case "$1" in
      "--outfile")
        echo "setting outfile: $2"
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
  cli_name="$(basename "${files[0]}")"
  out_file="${options[outfile]:-"$BCTILS_COMPILE_DIR/${cli_name}_complete.sh"}"

  log "$cli_name : ${lang} autogen for files : ${files[*]}"
  mkdir -p "$(dirname "$out_file")"

  # shellcheck disable=SC2094
  if ! bctils -autogen-lang py -autogen-src "${files[0]}" -autogen-outfile "$out_file" "$cli_name" "$args_verbatim" > "$out_file"
  then
    # shellcheck disable=SC2034
    bctils_err="bctils autogen failed"
    return
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
