#!/usr/bin/env bash
# last_modified_ms: {{.ModifiedTimeMs}}
# todo: add version metadata
# todo: add gotype to the top without effecting it somehow
{{/*gotype: shcomp2/pkg/lib.templateData*/}}

log () { echo -e "[$(date '+%T.%3N')] $*" >> ~/bashscript.log; }
log_everything () { if [[ "{{.Cli.CliNameClean}}" == "$1" ]]; then exec >> ~/bashscript.log; exec 2>&1; set -x; fi; }

{{.OperationsComment}}

__shcomp2_v2_autocomplete_{{.Cli.CliNameClean}} () {
  local -A subparsers={{ BashAssocNoQuote .ParserNameMap 2 }}

  # options
  {{range $parser := .Parsers -}}
  local -A _option_{{$parser.NameClean}}_name_map={{ BashAssocQuote $parser.OptionalsNameMap 2 }}
  local -a _option_{{$parser.NameClean}}_names={{ BashArray $parser.OptionalsNames 2 }}
  local -A _option_{{$parser.NameClean}}_data={{ BashAssocQuote $parser.OptionalsData 2 }}
  {{ end }}

  # arguments
  {{- range $parser := .Parsers -}}
  {{- if $parser.Subparsers }}
  {{/* subparsers are always the first and only positional */}}
  local _positional_{{$parser.NameClean}}_1_type="choices"
  local _positional_{{$parser.NameClean}}_1_choices={{- BashArray $parser.Subparsers 2 }}
  {{- else }}
  {{- range $pos := $parser.Positionals -}}
  {{- if and (ne $pos.NArgs.Max 0.0) (eq $pos.NArgs.Unique true) }}
  local -A _positional_{{$parser.NameClean}}_{{$pos.Number}}_used
  {{- end}}
  {{- if eq $pos.CompleteType "choices" }}
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_type="choices"
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_choices={{- BashArray $pos.Choices 2 }}
  {{- else if eq $pos.CompleteType "closure" }}
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_type="closure"
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_closure="{{ $pos.ClosureName }}"
  {{- end }}
  {{- end }}
  {{- end }}
  {{- end }}

  # todo: _get_comp_words_by_ref remove dependency on bash-completion repo
  # shellcheck disable=SC2034
  local cword_index previous_word words current_word
  _get_comp_words_by_ref -n = -n @ -n : -w words -i cword_index -p previous_word -c current_word

  log "complete: '$COMP_LINE'"

  # default add space after completion
  compopt +o nospace

  local -A used_options=()
  local carg_index=0
  local i=0 # skip first word
  local current_parser=""
  local current_parser_clean="baseparser"
  local completing_option_val=0
  while true; do
    i=$((i+1))
    if [[ -z "${words[$i]+set}" ]]; then break; fi
    word="${words[$i]}"

    # argument
    if [[ ! "$word" =~ ^'-' && "$i" -le "$cword_index" ]]; then
      carg_index=$((carg_index+1))
      {{ .NargsSwitch | indent 6 }}
    fi

    # option
    local -n option_data="_option_${current_parser_clean}_data"
    local -n option_map="_option_${current_parser_clean}_name_map"
    if [[ "$word" =~ ^'-' && "$i" -le "$cword_index" ]]; then
      if [[ ${#word} == 2 || -n ${option_map[$word]} ]]; then
        local reached_max=1
        if [[ -v "option_data[__narg_max__,$word]" ]]; then
          if [[ "${option_data["__narg_max__,$word"]}" == "inf" ]]; then
            reached_max=0
          else
            option_data["__narg_count__,$word"]=$((option_data["__narg_count__,$word"]+1))
            if [[ "${option_data["__narg_count__,$word"]}" -lt "${option_data["__narg_max__,$word"]}" ]]; then
              reached_max=0
              option_data[__narg_maxed__,$word]=1
            fi
          fi
        fi
        # todo: only add code if alternatives code is required
        local limit=99
        local idx=0
        local alt
        while true; do
          alt="${option_data["__alternatives__,$word,$idx"]}"
          if [[ $idx -ge $limit || -z "$alt" ]]; then break; fi
          idx=$((idx+1))
          option_map["$alt"]=0
        done
        option_data["__alternatives__,__used__,$word"]=1
        if ((i<=cword_index)); then
          if [[ "$reached_max" == 1 ]]; then
            used_options["$word"]=1
          fi
        fi
      elif [[ ${#word} -ge 2 || $cword_index == $i ]]; then
        # count option usage in merged opts
        local opt
        while IFS='' read -r -d '' -n 1 char; do
          if [[ $char != $'\n' && $char != '-' ]]; then
            opt="-$char"
            local reached_max=1
            if [[ -n ${option_data[__narg_max__,$opt]} ]]; then
              if [[ "${option_data["__narg_max__,$opt"]}" == "inf" ]]; then
                reached_max=0
              else
                option_data["__narg_count__,$opt"]=$((option_data["__narg_count__,$opt"]+1))
                if [[ "${option_data["__narg_count__,$opt"]}" -lt "${option_data["__narg_max__,$opt"]}" ]]; then
                  reached_max=0
                fi
              fi
            fi
            # todo: only add code if alternatives code is required
            local limit=99
            local idx=0
            local alt
            while true; do
              alt="${option_data["__alternatives__,$opt,$idx"]}"
              if [[ $idx -ge $limit || -z "$alt" ]]; then break; fi
              idx=$((idx+1))
              option_map["$alt"]=0
            done
            option_data["__alternatives__,__used__,$opt"]=1
            if [[ "$reached_max" == 1 ]]; then
              used_options["$opt"]=1
              option_map["$opt"]=0
            fi
          fi
        done <<< "$word"
      fi
    fi

    # current parser
    # todo: need a way to ensure subparser match isn't an arg or option value
    # todo: optimize based on "subparsers are invoked based on the value of the first positional argument..."
    if [[ "$i" -le "$cword_index" ]]; then
      if [[ -n "${current_parser}" ]]; then
        subparser_candidate="${current_parser},${word}"
      else
        subparser_candidate="${word}"
      fi
      if [[ -n "$subparser_candidate" && -n "${subparsers[$subparser_candidate]}" ]]; then
        current_parser="$subparser_candidate"
        current_parser_clean="${subparsers[$subparser_candidate]}"
        carg_index=0 # reset
      fi
    fi
  done

  if [[ "$carg_index" == 0 ]]; then
    carg_index=1  # todo: this is a hack. figure out how to properly get positional number based on line
  fi

  {{if .NargsSwitchHas }}
  # todo: remove need for this here
  {{ .NargsSwitch | indent 2 }}
  carg_index="$real_carg_index"
  {{end}}

  if [[ -z "$current_parser" ]]; then
    parser="{{.DefaultParserClean}}"
  else
    parser="$current_parser_clean"
  fi

  local choices_all=()
  local -n option_complete_data="_option_${parser}_data"
  if [[ -v option_complete_data[@] && -v "option_complete_data[__type__,$previous_word]" ]]; then
    # --option values
    # solve edge cases with mistaking positionals with options
    local option_name="$previous_word"
    local option_choices
    case "${option_complete_data[__type__,$option_name]}" in
      "choices")
        option_choices="${option_complete_data[__value__,$option_name]}"
        ;;
      "closure")
        local option_closure="${option_complete_data[__value__,$option_name]}"
        COMPREPLY=()
        declare -g shcomp2_CURRENT_WORD="$current_word"
        "$option_closure"
        option_choices="${COMPREPLY[*]}"
        COMPREPLY=()
        ;;
    esac
    mapfile -t COMPREPLY < <(compgen -W "${option_choices}" -- "$current_word")
  else
    # positionals
    local -n positional_complete_type="_positional_${parser}_${carg_index}_type"
    case "$positional_complete_type" in
      "choices")
        local -n positional_choices="_positional_${parser}_${carg_index}_choices"
        local -n positional_used="_positional_${parser}_${carg_index}_used"
        if [[ "${#positional_used[@]}" -gt 0 ]]; then
          for choice in "${positional_choices[@]}"; do
            if [[ -z "${positional_used[$choice]}" ]]; then
              choices_all+=("$choice")
            fi
          done
        else
          choices_all+=("${positional_choices[@]}")
        fi
        ;;
      "closure")
        local -n positional_closure="_positional_${parser}_${carg_index}_closure"
        COMPREPLY=()
        declare -g shcomp2_CURRENT_WORD="$current_word"
        "$positional_closure"
        choices_all+=("${COMPREPLY[@]}")
        COMPREPLY=()
        ;;
    esac

    local -n options_name_map="_option_${parser}_name_map"
    local -n options_name_seq="_option_${parser}_names"
    local -n options_name_dat="_option_${parser}_data"

    # options
    for name in "${options_name_seq[@]}"; do
      {{ if .Cli.Config.MergeSingleOpt }}
      local shortopt_merged shortopt_merged_appended=0 shortopt_left
      if [[ $current_word =~ -[^-].* ]]; then
        if [[ ${options_name_map[$current_word]} == 1 && ${#current_word} -gt 2 ]]; then
          # -longopt
          # todo: testcase for current word is -longopt or -lo
          :
        elif [[ -z "${options_name_dat[__type__,$current_word]}" ]]; then
          # for: -a -b -c -d
          # -a   => -ab -ac -ad
          # -ab  => -abc -abd
          # -abc => -abcd
          # todo: testcase for -abf where f takes a required value
          # todo: testcase for -abf where f takes an optional value
          # todo: looping over options seq twice
          shortopt_merged="$current_word"
          if [[ "${options_name_map[$name]}" == 1 && "${used_options[$name]}" != 1 && ${#name} == 2 ]]; then
            shortopt_merged+="${name##-}"
            shortopt_merged_appended=1
            options_name_map["$name"]=0
          fi
          shortopt_left="${current_word:0-1}"
        fi
      fi
      if [[ -n "$shortopt_merged" ]]; then
        if [[ $shortopt_merged_appended == 1 ]]; then
          choices_all+=("$shortopt_merged")
        fi
      else
        if [[ "${options_name_map[$name]}" == 1 && "${used_options[$name]}" != 1 ]]; then
          choices_all+=("$name")
        fi
      fi

      log "num2: ${#choices_all[@]}"
      log "choices_all2: ${choices_all[*]}"
      if [[ ${#choices_all[@]} == 1 && ${options_name_dat[__narg_nospace__,$name]} == 1 ]]; then
        log "NO SPACE ${options_name_dat[__narg_maxed__,$name]}"
        log "count: ${options_name_dat["__narg_count__,$name"]}"
        log "max  : ${options_name_dat["__narg_max__,$name"]}"
{{/*        compopt +o nospace*/}}
      elif [[ ${#choices_all[@]} == 1 ]]; then
        :
{{/*        compopt -o nospace*/}}
      fi
      {{ else }}
      {{/* no merging of short opts*/}}
      if [[ "${options_name_map[$name]}" == 1 && "${used_options[$name]}" != 1 ]]; then
        choices_all+=("$name")
      fi
      {{ end }}
    done

    log "choices_all: ${choices_all[*]}"

    mapfile -t COMPREPLY < <(compgen -W "${choices_all[*]}" -- "$current_word")
  fi
}

{{if .Cli.Config.IncludeSources}}
{{range .Cli.Config.IncludeSources -}}
source "{{.}}"
{{end}}
{{end}}

{{if .Cli.Config.AutogenReloadTriggers}}
__shcomp2_v2_autocomplete_autogen_reloader_{{.Cli.CliNameClean}} () {
  shcomp2 -reload-check <<'OEF'
    {{ .StringsJoin .Cli.OperationsReloadConfig 4 }}
OEF
  local return_code="$?"
  if [[ "$return_code" == 5 ]]; then
    source "{{.Cli.Config.Outfile}}" # source self to reload changes
  elif [[ "$return_code" != 0 ]]; then
    >&2 echo "reload-check failed: $return_code"
  fi

  __shcomp2_v2_autocomplete_{{.Cli.CliNameClean}}
}
complete -F __shcomp2_v2_autocomplete_autogen_reloader_{{.Cli.CliNameClean}} -o nospace "{{ .Cli.CliName }}"
{{else}}
# todo: add closure validation when sourcing
complete -F __shcomp2_v2_autocomplete_{{ .Cli.CliNameClean }} -o nospace "{{ .Cli.CliName }}"
{{end}}
