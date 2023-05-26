log () { echo -e "[$(date '+%T.%3N')] $*" >> ~/mybash.log; }
errmsg () { echo "$@" 1>&2; }

declare -g bctils_err=""

bctils_cli_register () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"

  log "registering: $cli_name"
  unset "__bctils_data_args_${cli_name_clean}"
  declare -g -a "__bctils_data_args_${cli_name_clean}"
  unset "__bctils_data_errors_${cli_name_clean}"
  declare -g -a "__bctils_data_errors_${cli_name_clean}"
}

bctils_cli_add () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"
  local -n bctils_data_args="__bctils_data_args_${cli_name_clean}"
  local -n bctils_data_errors="__bctils_data_errors_${cli_name_clean}"
  bctils_err=""

  local -a args=("$0")
  local -A options=()
  options[parser]=""
  while true; do
    if [[ -z "$1" ]]; then break; fi
    IFS="=" read -r arg arg_value <<< "$1"; shift
    case "$arg" in
      "--choices")
        options[choices]="$arg_value" ;;
      "-p")
        options[parser]="$arg_value" ;;
      "--") break ;;
      *) args+=("$arg") ;;
    esac
  done

  arg_type="${args[2]}"

  case "$arg_type" in
    "opt") arg_name="${args[3]}" ;;
    "pos") arg_name="" ;;
    *)
      error_str="bctils_cli_add error: second argument must be type opt or pos"
      errmsg "$error_str"
      bctils_data_errors+=("$error_str")
      bctils_err="second argument must be type opt or pos"
      return 1 ;;
  esac

  if [[ "$arg_type" == "opt" ]]; then
    arg_name="${args[3]}"
  elif [ "$arg_type" == "pos" ]; then
    arg_name=""
  fi

  if [[ -n "${options[parser]}" ]]; then
    part_parser=" -p=\"${options[parser]}\""
  else
    part_parser=""
  fi

  if [[ -n "${options[choices]}" ]]; then
    part_choices=" --choices=\"${options[choices]}\""
  else
    part_choices=""
  fi

  part_argtype="$arg_type"
  part_argname=" \"$arg_name\""

  printf -v arg_str '%s' "$part_argtype" "$part_argname" "$part_parser" "$part_choices"
  bctils_data_args+=("$arg_str")
}

bctils_cli_compile () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"
  bctils_err=""

  local -A options=()
  local -a args=()
  if ! TEMP=$(getopt -o '-h' --longoptions 'source' -- "$@"); then echo "failed to parse args"; exit 1; fi
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

  out_file="$(realpath "${args[1]:-"$BCTILS_COMPILE_DIR/${cli_name}_complete.sh"}")"

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

  test -f "$out_file" && chmod u+w "$out_file"
  if ! printf '%s\n' "${bctils_data_args[@]}" | bctils "$cli_name" > "$out_file"
  then
    # shellcheck disable=SC2034
    bctils_err="bctils compile failed in binary"
    return
  fi
  chmod u-w "$out_file"

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
