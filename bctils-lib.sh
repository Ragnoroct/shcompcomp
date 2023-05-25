log() { echo -e "[$(date '+%T.%3N')] bctils - $*" >> ~/mybash.log; }

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

  arg_type="$2"

  if [[ "$arg_type" == "opt" ]]; then
    arg_name="$3"
  else
    arg_name=""
  fi

  arg_str=$(
    printf '%s "%s"' "$arg_type" "$arg_name"
  )
  bctils_data_args+=("$arg_str")
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
