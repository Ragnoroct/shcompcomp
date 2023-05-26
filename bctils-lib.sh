log() { echo -e "[$(date '+%T.%3N')] $*" >> ~/mybash.log; }

bctils_cli_register () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"

  log "registering: $cli_name"
  unset "__bctils_data_args_${cli_name_clean}"
  declare -g -a "__bctils_data_args_${cli_name_clean}"
}

bctils_cli_add () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"
  local -n bctils_data_args="__bctils_data_args_${cli_name_clean}"

  local -a args=("$0")
  local -A options=()
  while true; do
    if [[ -z "$1" ]]; then break; fi
    IFS="=" read -r arg arg_value <<< "$1"; shift
    case "$arg" in
      "--choices")
        options[choices]="$arg_value" ;;
      "--") break ;;
      *) args+=("$arg") ;;
    esac
  done

  arg_type="${args[2]}"

  if [[ "$arg_type" == "opt" ]]; then
    arg_name="${args[3]}"
  else
    arg_name=""
  fi

  if [[ -n "${options[choices]}" ]]; then
    part_choices=" --choices=\"${options[choices]}\""
  else
    part_choices=""
  fi

  printf -v arg_str '%s "%s"%s' "$arg_type" "$arg_name" "$part_choices"
  bctils_data_args+=("$arg_str")
}

bctils_cli_compile () {
  local cli_name="$1"
  local cli_name_clean="${cli_name//[^[:alnum:]]/_}"

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
  test -f "$out_file" && chmod u+w "$out_file"
  if ! printf '%s\n' "${bctils_data_args[@]}" | bctils "$cli_name" > "$out_file"
  then
    exit
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
