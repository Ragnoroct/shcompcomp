#!/usr/bin/perl

my $root_dir;
if ($0 =~ /^\.\//) {
    $root_dir = "$ENV{'PWD'}";
} else {
    $root_dir = $0 =~ s#scripts\/clean_stack\.pl##r;
}
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

    if (/\.go:/g) {
        # on file line
        unless (s/$root_dir//) {
            $prev_line = "";
            $curr_line = "";
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
