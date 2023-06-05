#!/usr/bin/env bash

{{/*gotype: bctils.templateData*/}}

log () { echo -e "[$(date '+%T.%3N')] $*" >> ~/bashscript.log; }
log_everything () { if [[ "{{.Cli.CliNameClean}}" == "$1" ]]; then exec >> ~/bashscript.log; exec 2>&1; set -x; fi; }

{{.OperationsComment}}

__bctils_v2_autocomplete_{{.Cli.CliNameClean}} () {
  local -A subparsers={{ BashAssocNoQuote .ParserNameMap 2 }}

  # options
  {{range $parser := .Parsers -}}
  local _option_{{$parser.NameClean}}_names={{ BashArray $parser.OptionalsNames 2 }}
  local -A _option_{{$parser.NameClean}}_data={{ BashAssocQuote $parser.OptionalsData 2 }}
  {{ end }}

  # arguments
  {{range $parser := .Parsers -}}
  {{- range $pos := $parser.Positionals -}}
  {{- if eq $pos.CompleteType "choices" }}
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_type="choices"
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_choices={{- BashArray $pos.Choices 2 }}
  {{- else if eq $pos.CompleteType "closure" }}
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_type="closure"
  local _positional_{{$parser.NameClean}}_{{$pos.Number}}_closure="{{ $pos.ClosureName }}"
  {{- end}}
  {{ end }}
  {{- end }}

  # todo: fix _get_comp_words_by_ref dependency on bash-completion
  # shellcheck disable=SC2034
  local cword_index previous_word words current_word
  _get_comp_words_by_ref -n = -n @ -n : -w words -i cword_index -p previous_word -c current_word

  # default add space after completion
  compopt +o nospace

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
    # todo: optimize based on "subparsers are invoked based on the value of the first positional argument..."
    if [[ "$i" -le "$cword_index" ]]; then
      subparser_candidate="${current_parser}${word}"
      if [[ -n "${subparsers[$subparser_candidate]}" ]]; then
        current_parser="$subparser_candidate"
        current_parser_clean="${subparsers[$subparser_candidate]}"
        carg_index=1 # reset todo: why 1 intead of 0
      fi
    fi
  done

  if [[ -z "$current_parser" ]]; then
    parser="base"
  else
    parser="$current_parser_clean"
  fi

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
        declare -g BCTILS_CURRENT_WORD="$current_word"
        "$option_closure"
        option_choices="${COMPREPLY[*]}"
        COMPREPLY=()
        ;;
    esac
    mapfile -t COMPREPLY < <(compgen -W "${option_choices}" -- "$current_word")
  else
    local choices_all=()

    # positionals
    local -n positional_complete_type="_positional_${parser}_${carg_index}_type"
    case "$positional_complete_type" in
      "choices")
        local -n positional_choices="_positional_${parser}_${carg_index}_choices"
        choices_all+=("${positional_choices[@]}")
        ;;
      "closure")
        local -n positional_closure="_positional_${parser}_${carg_index}_closure"
        COMPREPLY=()
        declare -g BCTILS_CURRENT_WORD="$current_word"
        "$positional_closure"
        choices_all+=("${COMPREPLY[@]}")
        COMPREPLY=()
        ;;
    esac

    # options
    local -n choices_options="_option_${parser}_names"
    for i in "${!choices_options[@]}"; do
      choice_option="${choices_options[$i]}"
      if [[ "${used_options[$choice_option]}" != 1 ]]; then
        choices_all+=("${choices_options[$i]}")
      fi
    done

    mapfile -t COMPREPLY < <(compgen -W "${choices_all[*]}" -- "$current_word")
  fi
}

{{if .Cli.Config.SourceIncludes}}
{{range .Cli.Config.SourceIncludes -}}
source "{{.}}"
{{end}}
{{end}}

{{if .Cli.Config.ReloadTriggerFiles}}
__bctils_v2_autocomplete_autogen_reloader_{{.Cli.CliNameClean}} () {
  reload_files={{ BashArray .Cli.Config.ReloadTriggerFiles 2 }}
  reload_files_md5="$(md5sum "${reload_files[@]}")"
  cache_file="$HOME/.cache/bctils_autogen_md5_{{.Cli.CliNameClean}}"

  if [[
    "$__bctils_autogen_reload_cache_md5_{{.Cli.CliNameClean}}" != "$reload_files_md5" \
    && "$(cat "$cache_file" 2>/dev/null)" != "$reload_files_md5" \
  ]]
  then
    if {{.Cli.Config.AutoGenArgsVerbatim}}
    then
      source "{{.Cli.Config.AutoGenOutfile}}"
      echo "$reload_files_md5" > "$cache_file"
      __bctils_autogen_reload_cache_md5_{{.Cli.CliNameClean}}="$reload_files_md5"
    fi
  fi

  __bctils_v2_autocomplete_{{.Cli.CliNameClean}}
}
complete -F __bctils_v2_autocomplete_autogen_reloader_{{.Cli.CliNameClean}} -o nospace "{{ .Cli.CliName }}"
{{else}}
# todo: add closure validation when sourcing
complete -F __bctils_v2_autocomplete_{{ .Cli.CliNameClean }} -o nospace "{{ .Cli.CliName }}"
{{end}}
