#!/usr/bin/env bash

# todo: performance add debug conditional
log() { echo -e "$(date --iso-8601=ns) $*" >> ~/mybash.log; }

_bct_script_dir=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
declare -A commands_config
declare -A cli_auto_generate
declare -A bctils_v2_data # todo: each cli_name should have its own bctils_v2_data_$cli_name

bctils_v2_cli_register () {
  cli_name="$1"
  bctils_v2_data["$cli_name.reg"]="1"
  for i in "${!bctils_v2_data[@]}"; do
    if [[ "$i" =~ ^"$cli_name" ]]; then unset 'bctils_v2_data[$i]'; fi
  done
  bctils_v2_data[$cli_name,argument_count]=0
  bctils_v2_data[$cli_name,option_count]=0
  log "registered $cli_name"
}

bctils_v2_cli_add_argument () {
  local -a args=("$0")
  local -A options=()
  subparser=""
  while true; do
    if [[ -z "$1" ]]; then break; fi
    IFS="=" read -r arg arg_value <<< "$1"; shift
    case "$arg" in
      "--choices")
        options[choices]="$arg_value" ;;
      "-p")
        subparser="$arg_value" ;;
      "--") break ;;
      *) args+=("$arg") ;;
    esac
  done
  
  cli_name="${args[1]}"
  argument="${args[2]}"


  bctils_v2_data["$cli_name,$subparser,subparsername"]="$subparser"
  if [[ "$argument" =~ \-.*$ ]]; then
    local option_name="$argument"
    local option_index="${bctils_v2_data[$cli_name,$subparser,option_count]}"
    if [[ -z "$option_index" ]]; then
      option_index=0
    fi
    log "adding option $option_name ($option_index)"
    bctils_v2_data["$cli_name,$subparser,option,$option_index"]="$option_name"
    bctils_v2_data["$cli_name,$subparser,option,$option_name,name"]="$option_name"
    bctils_v2_data["$cli_name,$subparser,option,$option_name,index"]="$option_index"
    bctils_v2_data["$cli_name,$subparser,option,$option_name"]="$option_name"
    bctils_v2_data[$cli_name,$subparser,option_count]="$((option_index+1))"
    if [[ -v "options[choices]" ]]; then
      bctils_v2_data[$cli_name,$subparser,option,$option_name,choices]="${options[choices]}"
    fi
  else
    arg_number=$((bctils_v2_data[$cli_name,$subparser,argument_count] + 1))
    log "adding positional argument $arg_number with choices '${options[choices]}'"
    bctils_v2_data[$cli_name,$subparser,argument_count]="$arg_number"
    bctils_v2_data[$cli_name,$subparser,argument,$arg_number,number]="$arg_number"
    bctils_v2_data[$cli_name,$subparser,argument,$arg_number,choices]="${options[choices]}"
  fi
}

bctils_v2_cli_compile () {
  local profile_start=$(($(date +%s%N)/1000000))

  # NOTE: must use argument or --option=value
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

  cli_name="${args[0]}"
  out_file="$(realpath "${args[1]:-"$BCTILS_COMPILE_DIR/${cli_name}_complete.sh"}")"

  log "compiling completion script for $cli_name to $out_file"
  
  # shellcheck disable=SC2001
  cli_name_clean="$(echo "$cli_name" | sed 's/[^a-zA-Z0-9]//g')"
  
  mkdir -p "$(dirname "$out_file")"
  test -f "$out_file" && chmod u+w "$out_file"
  : > "$out_file"

  log "profiled first half: $((($(date +%s%N)/1000000)-profile_start))ms"
  local profile_start=$(($(date +%s%N)/1000000))
  
  # looping over needs to be more precise
  local strarr_arg_choices=()
  local strarr_options_vars=()
  local strarr_subparsers_assoc=()
  for k in "${!bctils_v2_data[@]}"; do
    if [[ "$k" =~ ^"$cli_name,".*",subparsername" ]]; then
      subparser="${bctils_v2_data[$k]}"
      # shellcheck disable=SC2001
      subparser_clean="$(echo "$subparser" | sed 's/[^a-zA-Z0-9]//g')"
      local options_array_str="local -A _options_subparser_${subparser_clean}_arr=("

      if [[ -n "$subparser" ]]; then
        strarr_subparsers_assoc+=("[$subparser]=\"$subparser_clean\"")
      fi
      
      local subparser_options_var_arr=()
      local option_index=-1
      while true; do
        option_index=$((option_index+1))
        if [[ -v "bctils_v2_data[$cli_name,$subparser,option,$option_index]" ]]; then
          local option_name="${bctils_v2_data[$cli_name,$subparser,option,$option_index]}"
          subparser_options_var_arr+=("$option_name")
          if [[ -v "bctils_v2_data[$cli_name,$subparser,option,$option_name,choices]" ]]; then
            options_array_str="$options_array_str"$'\n'"  [$option_name]=\"${bctils_v2_data[$cli_name,$subparser,option,$option_name,choices]}\""
          else
            options_array_str="$options_array_str"$'\n'"  [$option_name]=\"__NONE__\""
          fi
        else
          break
        fi
      done
      
      for j in "${!bctils_v2_data[@]}"; do
        if [[ "$j" =~ ^"$cli_name,$subparser,argument,".*",number" ]]; then
          arg_number="${bctils_v2_data[$j]}"
          if [[ -v "bctils_v2_data[$cli_name,$subparser,argument,$arg_number,choices]" ]]; then
            choices="${bctils_v2_data[$cli_name,$subparser,argument,$arg_number,choices]}"
            strarr_arg_choices+=("local _arg_choices_subparser_${subparser_clean}_${arg_number}=\"$choices\"")
          fi
        fi
      done
      options_array_str="$options_array_str"$'\n'")"
      strarr_options_vars+=("local _options_subparser_${subparser_clean}=\"${subparser_options_var_arr[*]}\"")
    fi
  done
  
  log "profiled str construct loop: $((($(date +%s%N)/1000000)-profile_start))ms"
  local profile_start=$(($(date +%s%N)/1000000))

  printf -v str_arg_choices "%s\n" "${strarr_arg_choices[@]}"
  printf -v str_options_vars "%s\n" "${strarr_options_vars[@]}"
  printf -v str_subparsers_assoc "%s\n" "${strarr_subparsers_assoc[@]}"
  
  template_vars=(
    cli_name_clean="$cli_name_clean"
    cli_name="$cli_name"
    str_options_vars:multiline="$str_options_vars"
    str_arg_choices:multiline="$str_arg_choices"
    str_subparsers_assoc:multiline="$str_subparsers_assoc"
    options_array_str:multiline="$options_array_str"
  )

  log "profiled printf vars + array creation: $((($(date +%s%N)/1000000)-profile_start))ms"
  local profile_start=$(($(date +%s%N)/1000000))

  cat <<'EOF' | __bctils_dedent >> "$out_file"
    #!/usr/bin/env bash
    log() { echo -e "$(date --iso-8601=ns) $*" >> ~/mybash.log; }
    
    __bctils_v2_autocomplete_{{cli_name_clean}} () {
        {{str_options_vars:multiline}}
        {{str_arg_choices:multiline}}
        {{options_array_str:multiline}}
        local -A subparsers=(
          {{str_subparsers_assoc:multiline}}
        )
        
        # shellcheck disable=SC2034
        local cword_index previous_word words current_word
        _get_comp_words_by_ref -n = -n @ -n : -w words -i cword_index -p previous_word -c current_word
        
        local -A used_options=()
        local carg_index=0
        local i=-1
        local current_parser=""
        local current_parser_clean=""
        local completing_option_val=0
        while true; do
          i=$((i+1))
          if [[ -z "${words[$i]}" ]]; then break; fi
          word="${words[$i]}"
          
          # argument
          if [[ ! "$word" =~ ^'-' && "$i" -lt "$cword_index" ]]; then
            carg_index=$((carg_index+1))
          fi

          # option
          if [[ "$word" =~ ^'-' && "$i" -le "$cword_index" ]]; then
            if ((i<=cword_index)); then
              used_options["$word"]=1
            fi
          fi

          # current parser
          # todo: need a way to ensure subparser match isn't an arg or option value
          if [[ "$i" -le "$cword_index" ]]; then
            subparser_candidate="${current_parser}${word}"
            if [[ -v "subparsers[$subparser_candidate]" ]]; then
              current_parser="$subparser_candidate"
              current_parser_clean="${subparsers[$subparser_candidate]}"
              carg_index=1 # resets (todo: why 1 intead of 0)
            fi
          fi
        done

        # todo: local -n options_arr requires bash 4.3
        # todo: think of edge cases with arguments and mistaking
        local -n options_arr="_options_subparser_${current_parser_clean}_arr"
        if [[ -v "options_arr[$previous_word]" && "${options_arr[$previous_word]}" != "__NONE__" ]]; then
          # completing --option value
          local option_choices="${options_arr[$previous_word]}"
          mapfile -t COMPREPLY < <(compgen -W "${option_choices}" -- "$current_word")
        else
          # completing command option or arg
          local choices_all=()

          # add arg choices
          local choices_args=()
          local choices_args_key="_arg_choices_subparser_${current_parser_clean}_${carg_index}"
          if [[ -v "$choices_args_key" ]]; then
            IFS=' ' read -r -a choices_args <<< "${!choices_args_key}"
          fi
          choices_all=("${choices_all[@]}" "${choices_args[@]}")

          # add options
          local choices_options=()
          local options_key="_options_subparser_${current_parser_clean}"
          if [[ -v "$options_key" ]]; then
            IFS=' ' read -r -a choices_options <<< "${!options_key}"
          fi
          for i in "${!choices_options[@]}"; do
            choice_option="${choices_options[$i]}"
            if [[ "${used_options[$choice_option]}" != 1 ]]; then
              choices_all+=("${choices_options[$i]}")
            fi
          done
          
          mapfile -t COMPREPLY < <(compgen -W "${choices_all[*]}" -- "$current_word")
        fi
    }

    complete -F __bctils_v2_autocomplete_{{cli_name_clean}} "{{cli_name}}"
EOF

  log "profiled OEF into dedent: $((($(date +%s%N)/1000000)-profile_start))ms"
  local profile_start=$(($(date +%s%N)/1000000))

  __bctils_format_template "$out_file" "${template_vars[@]}"

  log "profiled template: $((($(date +%s%N)/1000000)-profile_start))ms"
  local profile_start=$(($(date +%s%N)/1000000))

  chmod u-w "$out_file"

  if [[ "${options["source"]}" == "1" ]]; then
    log "sourcing $out_file"
    # shellcheck disable=SC1090
    source "$out_file"
  fi

  # clean state
  for i in "${!bctils_v2_data[@]}"; do
    if [[ "$i" =~ ^"$cli_name" ]]; then unset 'bctils_v2_data[$i]'; fi
  done

  log "profiled source + chmod + clean state: $((($(date +%s%N)/1000000)-profile_start))ms"
}

# https://github.com/git/git/blob/master/contrib/completion/git-completion.bash
bashcompletils_autocomplete () {
    profile_start=$(date +%s.%N)

    # shellcheck disable=SC2034
    local cword_index previous_word words current_word
    COMPREPLY=()

    # allow characters =@: to not split words
    _get_comp_words_by_ref -n = -n @ -n : -w words -i cword_index -p previous_word -c current_word

    current_parser="${words[0]}"

    if [[ -v "cli_auto_generate[$current_parser]" ]]; then
      local additional_files=()
      local i=0; while true; do
        i=$((i+1))
        if [[ -v "cli_auto_generate[$cli_name.$i]" ]]; then
          additional_files+=("${cli_auto_generate[$cli_name.$i]}")
        else
          break
        fi
      done
      static_parser_python_refresh "${cli_auto_generate[$current_parser]}" "${additional_files[@]}"
    fi

    pos_idx=0
    nargs=1
    local nargs_in_middle=false
    local nargs_current=()
    local words_index=-1
    local words_length="${#words[@]}"
    for word in "${words[@]}"; do
      words_index=$((words_index+1))
      local key_nargs="$current_parser%pos_args%meta%nargs,$pos_idx"
      if [[ -v "commands_config[$key_nargs]" ]]; then
        nargs="${commands_config[$key_nargs]}"
        if [[ "$nargs" == "*" && "$nargs_in_middle" == false ]]; then
          nargs_current=()
          nargs_in_middle=true # todo: reset this somehow
        fi
      fi
    
      if [[ "$word" != "${words[0]}" ]]; then
        case "$word" in
          -|"") ;;
          *) 
            if [[ "$nargs" != "*" ]]; then
              if [[ "$words_index" -ge "$((words_length-1))" && "$word" != "" ]]; then
                : # todo: do this without if else
              else
                pos_idx=$((pos_idx + 1))
              fi
            fi
            nargs_current+=("$word") # adding parser command instead of arguments
            ;; # todo: handle options with arguments. they shouldn't count as positional arguments
        esac
      fi

      # todo: Do actual for i loop where we can keep track of actual indexes. if two words are the same as the last word but the first one isn't the last word it will trigger here
      if [[ "$word" != "${words[-1]}" ]]; then
        potential_index="${current_parser}.${word}"
        if [[ "${commands_config[$potential_index%static_values%,length]}" -gt 0 ]]; then
          current_parser="$potential_index"
          pos_idx=0
        fi
        if [[ "${commands_config[$potential_index%options%static_values%,length]}" -gt 0 ]]; then
          current_parser="$potential_index"
          pos_idx=0
        fi
      fi
    done

    local key_closure="$current_parser%pos_args%meta%closure,$pos_idx"
    if [[ "${commands_config[$key_closure]}" != "" ]]; then
      closure_func="${commands_config[$key_closure]}"
      "$closure_func"
    else
      # positionals
      static_values_str="${commands_config[$current_parser%static_values%,$pos_idx]}"
      read -r -a static_values_array <<< "$static_values_str"
      if [[ "${#static_values_array[@]}" -gt 0 ]]; then
        # todo: this needs to be more robust
        if [[ "$nargs" == "*" ]]; then
          new_static_values_array=()
          # todo: double look performance is bad
          local found
          for val in "${static_values_array[@]}"; do
            found=false
            for narg_already in "${nargs_current[@]}"; do
              if [[ "$val" == "$narg_already" ]]; then
                found=true
                break
              fi
            done
            if [[ "$found" == "false" ]]; then
              new_static_values_array+=("$val")
            fi
          done
          static_values_array=("${new_static_values_array[@]}")
        fi
        mapfile -t COMPREPLY < <(compgen -W "${static_values_array[*]}" -- "$current_word")
      # options
      elif [[ "${#COMPREPLY}" == "0" ]]; then
        # todo: for now index value will be 0 hardcoded
        local static_values_array=()
        local i=0
        while true; do
          static_value="${commands_config[$current_parser%options%static_values%,$i]}"
          if [[ -n "$static_value" ]]; then
            # todo: remove options already used
            static_values_array+=("$static_value")
          else
            break
          fi
          i=$((i+1))
        done
        if [[ "${#static_values_array[@]}" -gt 0 ]]; then
          mapfile -t COMPREPLY < <(compgen -W "${static_values_array[*]}" -- "$current_word")
        fi
      fi
    fi

    log "profiled: $( echo "$(date +%s.%N) - $profile_start" | bc -l )s"
}

add_positional_argument () {
  local closure=""
  local nargs=1

  # NOTE: --longoptions does not support --opt2 val2. It has to be --opt2=val2  
  local args=()
  if ! TEMP=$(getopt -o '-h' --longoptions 'nargs::,closure::' -- "$@"); then echo "failed to parse args"; exit 1; fi
  eval set -- "$TEMP"; unset TEMP
  while true; do
    case "$1" in
      "--nargs") 
        case "$2" in
          [0-9]|"*") nargs="$2"; shift 2 ;;
          *) echo "invalid value for --nargs '$2'"; exit 1 ;;
        esac
        ;;
      "--closure")
        closure="$2"
        shift 2
        ;;
      "--") break ;;
      *) args+=("$1"); shift ;;
    esac
  done

  parser="${args[0]}"
  static_values="${args[1]}"

  length_key="$parser%static_values%,length"
  if [[ ! -v "commands_config[$length_key]" ]]; then
    commands_config[$length_key]=0
  fi
  
  local pargs_idx="${commands_config[$length_key]}"
  commands_config["$parser%static_values%,$pargs_idx"]="$static_values"

  local key_closure="$parser%pos_args%meta%closure,$pargs_idx"
  commands_config["$key_closure"]=$closure

  local key_nargs="$parser%pos_args%meta%nargs,$pargs_idx"
  commands_config["$key_nargs"]="$nargs"
  
  pargs_idx=$((pargs_idx+1))
  commands_config[$length_key]="$pargs_idx"
}

add_option_argument () {
  length_key="$1%options%static_values%,length"
  if [[ ! -v "commands_config[$length_key]" ]]; then
    commands_config[$length_key]=0
  fi
  
  local pargs_idx="${commands_config[$length_key]}"
  commands_config["$1%options%static_values%,$pargs_idx"]="$2"

  pargs_idx=$((pargs_idx+1))
  commands_config[$length_key]="$pargs_idx"
}

reset_config () {
  cli_name="$1"
  for i in "${!commands_config[@]}"; do
    if [[ "$i" =~ ^"$cli_name" ]]; then unset 'commands_config[$i]'; fi
  done
}

static_parser_python_refresh () {
  target_file="$1"; shift
  cli_name="$(basename "$target_file")"
  log md5sum5 "$target_file" "$@"
  md5=$(md5sum "$target_file" "$@")
  md5_cache_array="${commands_config[$cli_name%refresh_cache%md5]}"
  cache_dir="$HOME/.cache/bashcompletils"
  cache_content_file="$cache_dir/$cli_name.content"
  cache_md5_file="$cache_dir/$cli_name.md5"

  # todo: add file caching and reload when array is empty
  if [[ "$md5" != "$md5_cache_array" ]]; then
    md5_cache_file="$(cat "$cache_md5_file" 2> /dev/null)"
    # todo: "$md5" == "$(md5sum "$cache_content_file" 2> /dev/null)"
    # add some security check on cached content we're just evaling
    if [[ "$md5" == "$md5_cache_file" ]]; then
      log "refreshing cache from file for: $cli_name $target_file"
      for i in "${!commands_config[@]}"; do
        if [[ "$i" =~ ^"$cli_name" ]]; then unset 'commands_config[$i]'; fi
      done
      commands_output=$(cat "$cache_content_file")
      eval "$commands_output"
      commands_config["$cli_name%refresh_cache%md5"]="$md5"
    else
      log "refreshing cache from script for: $cli_name $target_file"
      for i in "${!commands_config[@]}"; do
        if [[ "$i" =~ ^"$cli_name" ]]; then unset 'commands_config[$i]'; fi
      done
      debug_commands_out=""
      if commands_output=$("$_bct_script_dir/bashcompletils-lib.py" --cliname "$cli_name" "$target_file" "$@"); then
        debug_commands_out="======="
        debug_commands_out="$debug_commands_out"$'\n'"==== $cli_name : $target_file "$'\n'"$commands_output"
      
        log "\n$debug_commands_out\n======="
        eval "$commands_output"
        commands_config["$cli_name%refresh_cache%md5"]="$md5"
        mkdir -p "$cache_dir"
        printf "%s\n" "$md5" > "$cache_md5_file"
        printf "%s\n" "$commands_output" > "$cache_content_file"
      fi
    fi
  fi
}

register_python_auto_gen () {
  target_file="$1"; shift
  
  test -f "$target_file" || { echo "register_python_auto_gen error: invalid file $target_file"; return; }
  cli_name="$(basename "$target_file")"
  log "registering: $cli_name=$target_file"
  cli_auto_generate["$cli_name"]="$target_file"
  
  i=0; for additional_file in "$@"; do i=$((i+1))
    log "registering: $cli_name+=$additional_file"
    cli_auto_generate["$cli_name.$i"]="$additional_file"
  done

  # mainly for debugging
  echo "$target_file $*" > "/tmp/bashcompletils-$cli_name-initial-args"
  cli_auto_generate["initial_args,$cli_name"]="$target_file $*"   

  complete -F bashcompletils_autocomplete "$cli_name"
}

__bashcompletils_branchname () {
  # shellcheck disable=SC2034
  local cword_index previous_word words current_word
  _get_comp_words_by_ref -n = -n @ -n : -w words -i cword_index -p previous_word -c current_word
  COMPREPLY=()

  all_branches=$(git for-each-ref --format='%(refname:short)' refs/heads/)
  IFS=$'\n' read -d '' -r -a all_branches_array <<< "$all_branches"
  mapfile -t COMPREPLY < <(compgen -W "${all_branches_array[*]}" -- "$current_word")
}

array_cheatsheet () {
  echo "just for reference"
  key="keyname"
  declare -A arr

  # associative: assign values
  arr["literal"]="value"
  arr["$key,literal"]="value"

  # delete all keys with prefix in associative array
  for i in "${!arr[@]}"; do
    if [[ "$i" =~ ^"$cli_name" ]]; then unset 'arr[$i]'; fi
  done
}

__bctils_dedent () {
  LF=$'\n'
  buffer=""
  min_indent=999
  while IFS='' read -r line; do
    if [[ -n "${line// }" ]]; then
      # shellcheck disable=SC2308
      cur_indent=$(expr match "$line" " *")
      min_indent=$((min_indent<cur_indent ? min_indent : cur_indent))
    fi
    buffer="$buffer$line$LF"
  done

  # shellcheck disable=SC2183
  printf -v indent_str '%*s' "$min_indent"
  
  new_buffer=""
  while IFS='' read -r line; do
    new_buffer="${new_buffer}${line#"$indent_str"}${LF}"
  done <<< "$buffer"
  new_buffer="${new_buffer%% }"
  new_buffer="${new_buffer%%$'\n'}"
  printf "%s" "$new_buffer"
}

__bctils_format_template () {
  template_file="$1"; shift

  for arg_str in "$@"; do
    IFS="=" read -r -d $'\0' var_name var_value <<< "$arg_str"
    if [[ "$var_name" =~ .*:multiline ]]; then
      # special indenting for multiline stuff
      while IFS="" read -r line_match; do
        # shellcheck disable=SC2308
        indent=$(expr match "$line_match" " *")
        # shellcheck disable=SC2183
        printf -v indent_str '%*s' "$indent"
        first_line=1
        indented_lines_buffer=""
        while IFS="" read -r var_value_line; do
          if [[ -z "${var_value_line// }" ]]; then continue; fi
          if [[ "$first_line" == 1 ]]; then
            first_line=0
            indented_lines_buffer="${indent_str}${var_value_line}"
          else
            indented_lines_buffer="$indented_lines_buffer"$'\n'"${indent_str}${var_value_line}"
          fi
        done <<< "$var_value"
        indented_lines_buffer=${indented_lines_buffer//$'\n'/\\n} # escape \n
        sed -i "s/$line_match/$indented_lines_buffer/g" "$template_file"
      done < <(sed -n "/{{$var_name}}/p" "$template_file")
    else
      var_value=${var_value%$'\n'}      # strip trailing \n
      var_value=${var_value//$'\n'/\\n} # escape \n
      sed -i "s/{{$var_name}}/$var_value/g" "$template_file"
    fi
  done
}
