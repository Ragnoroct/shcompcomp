#!/usr/bin/perl

my $root_dir = $0 =~ s#scripts\/clean_stack\.pl##r;
my $indent_len = 0;
my $leading_spaces = "";
my $in_stack = 0;
my $prev_line = "";
my $curr_line = "";
my $build_failed = 0;
while (<>) {
    if (/\[build failed\]/) {
        $build_failed = 1;
    }

    # buffer two lines at a time
    if ($curr_line ne "") {
        $prev_line = $curr_line;
    }
    $curr_line = "$_";

    if ($_ =~ "test panicked:") {
        $in_stack = 1;
        ($leading_spaces) = /^(\s*)/;
        $indent_len = length $leading_spaces;
    } elsif ($in_stack eq 1) {
        ($leading_spaces) = /^(\s*)/;
        if ((length $leading_spaces) < $indent_len) {
            $in_stack = 0;
        } else {
            # inside stacktrace
            if (/\.go:/g) {
                # on file line
                unless (s/$root_dir//) {
                    $prev_line = "";
                    $curr_line = "";
                }
            } elsif (/goroutine.*?\[running\]:/) {
                $curr_line = "";
            }
        }
    }

    if ($prev_line ne "") {
        print $prev_line;
        print $curr_line;
        $prev_line = "";
        $curr_line = "";
    }
}

print $prev_line;
print $curr_line;

if ($build_failed eq 1) {
    exit 2;
} else {
    exit 1;
}
