#!/usr/bin/env python3
from logging import basicConfig, getLogger
from os import environ
from sys import argv, exit as sys_exit, stderr as sys_stderr, stdout as sys_stdout
from pathlib import Path
from ast import (
    parse as ast_parse, 
    walk as ast_walk, 
)
import ast
import tokenize
from tokenize import COMMENT
from argparse import ArgumentParser
from io import StringIO

basicConfig(
    filename=Path.home() / "bashscript.log",
    format="%(asctime)s.%(msecs)03d %(msg)s",
    datefmt="%Y-%m-%dT%H:%M:%S",
    force=True
)
logger = getLogger(__name__)
logger.setLevel(environ.get("LOGLEVEL", "INFO"))

def main():
    parser = ArgumentParser()
    parser.add_argument("target_scripts", nargs="+")
    parser.add_argument("--cliname", required=True)
    argparse_args = parser.parse_args()
    
    logger.info(argparse_args.target_scripts)
    source_string = ""
    for target in argparse_args.target_scripts:
        logger.info(f"parsing {target}")
        source_string += Path(target).read_text() + "\n"
    
    source_file_io = StringIO(source_string)
    source_file_io.readline
    
    # this is so hacky to handle multiple files
    output_lines = parse_script(argparse_args.cliname, argparse_args.target_scripts[0], source_file_io)

    for line in output_lines:
        sys_stdout.write(line + "\n")
    sys_stdout.flush()
    sys_exit(0)


def parse_script(cli_name, source_filename, source_file_io):
    add_argument_calls = []
    sub_parser_map = {}
    incrementer = -1

    comment_annotations = {}
    tokens = tokenize.generate_tokens(source_file_io.readline)
    for t in tokens:
        if t.type == COMMENT and t.string.removeprefix("#").lstrip().startswith("bashcompletils"):
            line_number = t.start[0]
            comment_annotations[line_number] = {
                "token": t,
            }

    source_file_io.seek(0)
    script_ast = ast_parse(source=source_file_io.read().encode(), filename=source_filename)
    for node in ast_walk(script_ast):
        if False: pass
        # assignment
        elif isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name):
                    if isinstance(node.value, ast.Call):
                        node_call = node.value
                        if hasattr(node_call.func, "attr") and node_call.func.attr == "add_parser":
                            func_name = node_call.func.attr
                            args = []
                            for node_arg in node_call.args:
                                if isinstance(node_arg, ast.Constant):
                                    args.append(node_arg.value)
                                else: raise Exception("todo: non-constants")
                            sub_parser_map[target.id] = {
                                "name": args[0]
                            }
        # non-assignment
        if isinstance(node, ast.Call):
            node_call = node
            if hasattr(node_call.func, "attr") and node_call.func.attr == "add_argument":
                comment_annotation = ""
                try:
                    sameline_comment_tokeninfo = comment_annotations[node_call.end_lineno]
                    sameline_comment_str = sameline_comment_tokeninfo["token"].string
                    sameline_comment_str = sameline_comment_str.lstrip().lstrip("#").strip()
                    if sameline_comment_str.startswith("bashcompletils"):
                        comment_annotation = sameline_comment_str.replace("bashcompletils", "").strip()
                except KeyError:
                    pass
                
                node_parent_parser = node_call.func.value.id
                func_name = node_call.func.attr
                args = []
                kwargs = {}
                for node_arg in node_call.args:
                    if isinstance(node_arg, ast.Constant):
                        args.append(node_arg.value)
                    else: raise Exception("todo: non-constants")
                for node_kwarg in node_call.keywords:
                    kwarg_name = node_kwarg.arg
                    kwarg_node_value = node_kwarg.value
                    kwarg_value = None
                    if hasattr(kwarg_node_value, "elts"):
                        kwarg_value = []
                        for list_value in kwarg_node_value.elts:
                            if isinstance(list_value, ast.Constant):
                                kwarg_value.append(list_value.value)
                            elif isinstance(list_value, ast.List) and len(list_value.elts) == 0:
                                pass    # empty list to allow ignored values
                            else: raise Exception("todo: unhandled datatype as choice")
                    elif isinstance(kwarg_node_value, ast.Constant):
                        kwarg_value = kwarg_node_value.value
                    else: raise Exception("todo")
                    kwargs[kwarg_name] = kwarg_value
                add_argument_calls.append({
                    "parent_command_nodename": node_parent_parser,
                    "func_nodename": func_name,
                    "args": args,
                    "kwargs": kwargs,
                    "comment_annotation": comment_annotation,
                })
            # add subparser results that don't assign to any variable
            if hasattr(node_call.func, "attr") and node_call.func.attr == "add_parser":
                incrementer += 1
                func_name = node_call.func.attr
                args = []
                for node_arg in node_call.args:
                    if isinstance(node_arg, ast.Constant):
                        args.append(node_arg.value)
                    else: raise Exception("todo: non-constants")
                already_added = False
                for key in sub_parser_map:
                    if sub_parser_map[key]["name"] == args[0]:
                        already_added = True
                if not already_added:
                    sub_parser_map[incrementer] = {
                        "name": args[0]
                    }
        elif isinstance(node, ast.Name):
            # Assign
            pass
            # print("name:", node.id, "value:", node.value)
        elif isinstance(node, ast.Attribute):
            pass
            # print("attr:", node.attr)
        elif isinstance(node, ast.FunctionDef):
            pass
    
    for add_arg in add_argument_calls:
        parent_cmd = sub_parser_map[add_arg["parent_command_nodename"]]
        add_arg["parent_command"] = parent_cmd

    target_script_name = cli_name

    output_lines = []

    sub_commands = []
    for sub_parser_key in sub_parser_map:
        sub_parser = sub_parser_map[sub_parser_key]
        if sub_parser.get("parent") is None:
            sub_commands.append(sub_parser["name"])
    output_lines.append(f'''add_positional_argument "{target_script_name}" "{' '.join(sub_commands)}"''')

    for add_argument_call in add_argument_calls:
        arg_name = add_argument_call["args"][0]
        if arg_name.startswith("-"):
            current_parser = [target_script_name]
            parent_command_nodename = add_argument_call["parent_command_nodename"]
            while parent_command_nodename is not None:
                parent_parser = sub_parser_map[parent_command_nodename]
                current_parser.append(parent_parser["name"])
                parent_command_nodename = parent_parser.get("parent")
            
            output_lines.append(f'''add_option_argument "{current_parser_str}" "{arg_name}"''')
        else:
            # todo: remove none as workaround as choices but empty. use default=[] or something I think
            current_parser = [target_script_name]
            parent_command_nodename = add_argument_call["parent_command_nodename"]
            while parent_command_nodename is not None:
                parent_parser = sub_parser_map[parent_command_nodename]
                current_parser.append(parent_parser["name"])
                parent_command_nodename = parent_parser.get("parent")
            
            current_parser_str = ".".join(current_parser)
            
            if add_argument_call["kwargs"].get("choices"):
                choices = add_argument_call["kwargs"].get("choices")
                output_lines.append(f'''add_positional_argument "{current_parser_str}" "{' '.join(choices)}"''')
            elif add_argument_call["comment_annotation"] != "":
                for args in add_argument_call["comment_annotation"].split():
                    key_value_split = args.split("=")
                    if key_value_split[0] == "func":
                        func_name = key_value_split[1]
                        if func_name == "branchname":
                            output_lines.append(f'''add_positional_argument "{current_parser_str}" --closure="__bashcompletils_branchname"''')
    return output_lines
    # add_positional_argument "example_cli" "pio channel deploy release streamermap"
    # add_positional_argument "example_cli.channel" "deploy rm"
    # add_option_argument "example_cli.channel.rm" "--all"
    # add_option_argument "example_cli.channel.deploy" "--print-only"
    # add_option_argument "example_cli.channel.deploy" "-v"
    # add_positional_argument "example_cli.release" --closure=_example_cli_branch_autocomplete
    # add_positional_argument "example_cli.release" "qa dev rob" --nargs="*"
    # add_positional_argument "example_cli.streamermap" "push pull"
    # add_positional_argument "example_cli.streamermap.push" "qa dev rob" --nargs="*"
    # complete -F bashcompletils_autocomplete "example_cli"


if __name__ == "__main__":
    main()
