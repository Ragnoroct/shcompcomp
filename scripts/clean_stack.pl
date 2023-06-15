#!/usr/bin/perl

my $root_dir;
if ($0 =~ /^\.\//) {
    $root_dir = "$ENV{'PWD'}";
} else {
    $root_dir = $0 =~ s#scripts\/clean_stack\.pl##r;
}
my $first_line = 1;
my $prev_line = "";
my $curr_line = "";
my $build_failed = 0;
while (<>) {
    if (/\[build failed\]/) {
        $build_failed = 1;
    }

    # buffer two lines at a time
    if ($first_line eq 0) {
        $prev_line = $curr_line;
    } else {
        $first_line = 0;
    }
    $curr_line = "$_";

    if (/^\s*\/.*\.go:/g) {
        # on file line
        unless (s/$root_dir//) {
            if ($prev_line =~ /\.go/) {
                $curr_line = "";
            } else {
                $prev_line = "";
                $curr_line = "";
            }
        }
    } elsif (/goroutine.*?\[running\]:/) {
        $prev_line = "";
        $curr_line = "\n";
    }

    if ($prev_line ne "") {
        print $prev_line;
    }
}

print $prev_line;
print $curr_line;

if ($build_failed eq 1) {
    exit 2;
} else {
    exit 1;
}
