# shcompcomp
Generate command completion scripts using simple configs

**supported shells**
- [x] bash


### Examples
#### List of options
```bash
shcomp2 - > ~/.bash_completion.d/examplecli.bash <<EOF
cfg cli_name=examplecli
opt --help
opt -h
EOF

# behavior
--help -h
$ examplecli [TAB]
-h
$ examplecli --help [TAB]
```

#### Positional with choices
```bash
shcomp2 - > ~/.bash_completion.d/examplecli.bash <<EOF
cfg cli_name=examplecli
pos --choices="do_thing do_other nothing"
EOF

# behavior
do_thing do_other nothing
$ examplecli [TAB]
do_thing do_other
$ examplecli do_ [TAB]
```
