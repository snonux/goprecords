#!/usr/bin/env raku

use v6.d;

enum Category <Host Kernel KernelMajor KernelName>;
enum Metric <Boots Uptime Score Downtime Lifespan>;
enum OutputFormat <Plaintext Markdown Gemtext>;
subset MetricSubset of Metric where * ne any (Downtime, Lifespan);

our constant %DESCRIPTION = {
    Boots => 'Boots is the total number of host boots over the entire lifespan.',
    Uptime => 'Uptime is the total uptime of a host over the entire lifespan.',
    Downtime => 'Downtime is the total downtime of a host over the entire lifespan.',
    Lifespan => 'Lifespan is the total uptime + the total downtime of a host.',
    Score => 'Score is calculated by combining all other metrics.',
};

our UInt constant DAY = 1 * 24 * 3600;
our UInt constant MONTH = 30 * DAY;

class Epoch {
  has UInt $.value is required;

  submethod new (UInt $value) { self.bless: :$value }

  method human-duration returns Str {
    my DateTime \dt .= new: Instant.from-posix: $!value;
    "{dt.year-1970} years, {dt.month} months, {dt.day} days";
  }

  method human-date returns Str {
    DateTime.new(Instant.from-posix: $!value).yyyy-mm-dd;
  }

  method newer-than(UInt:D \limit --> Bool) {
    (DateTime.now - DateTime.new: Instant.from-posix: $!value) < limit * DAY;
  }
}

class Aggregate {
  has Str $.name is required;
  has UInt $.uptime;
  has UInt $.first-boot;
  has UInt $.last-seen;
  has UInt $.boots;

  method new (Str:D $name) { self.bless: :$name }

  method add-record(Str:D :$uptime, Str:D :$boot-time) {
      my $last-seen = $uptime + $boot-time;
      $!uptime += $uptime;
      $!boots++;

      $!first-boot = +$boot-time if not defined $!first-boot or $!first-boot > $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen or $!last-seen < $last-seen;
  }

  method meta-score returns UInt {
    UInt((($!uptime * 2) + ($!boots * DAY) + (self.is-active ?? MONTH !! 0))/1000000)
  }

  method is-active(UInt:D \limit = 90 --> Bool) {
    Epoch.new($!last-seen).newer-than: limit;
  }
}

class HostAggregate is Aggregate {
  method lifespan returns UInt { $.last-seen - $.first-boot }
  method downtime returns UInt { self.lifespan - $.uptime }
  method meta-score returns UInt { UInt(self.downtime / 2000000) + callsame }
}

class Aggregator {
  has Hash %!aggregates = { Host => {}, Kernel => {}, KernelName => {}, KernelMajor => {} }
  has Str $.stats-dir is required;

  submethod new (Str:D $stats-dir) { self.bless: :$stats-dir }

  method aggregate () {
    self!add-file: $_ for dir $!stats-dir, test => { /.records$/ };
    return %!aggregates;
  }

  method !add-file(IO::Path:D $file) {
    return if $file.s == 0;
    my $host = $file.IO.basename.split('.').first;

    die "Record file for $host already processed - duplicate inputs?"
      if %!aggregates<Host>{$host}:exists;

    %!aggregates<Host>{$host} = HostAggregate.new: $host;
    self!add-line: :line($_), :$host for $file.IO.lines;
  }

  method !add-line(Str:D :$line, Str:D :$host) {
    my ($uptime, $boot-time, $os) = $line.trim.split: ':';
    my $uname = $os.split(' ').first;
    my $os-major = "$uname {$os.split(' ')[1].split('.').first}...";

    %!aggregates<Kernel>{$os} //= Aggregate.new: $os;
    %!aggregates<KernelName>{$uname} //= Aggregate.new: $uname;
    %!aggregates<KernelMajor>{$os-major} //= Aggregate.new: $os-major;

    .add-record: :$uptime, :$boot-time
      for %!aggregates<Host>{$host}, %!aggregates<Kernel>{$os},
          %!aggregates<KernelName>{$uname}, %!aggregates<KernelMajor>{$os-major};
  }
}

role OutputHelper {
  has OutputFormat $.output-format is required;
  has UInt $.header-indent = 1;

  method output-header {
    ($.output-format ~~ any Markdown, Gemtext) ?? '#' x $.header-indent ~ ' ' !! ''
  }

  method output-trim(Str \str, UInt \line-limit --> Str) {
    if $.output-format ~~ Plaintext and str.chars > line-limit {
      return join '', gather {
        my $chars = 0;
        for str.split(' ') -> \word {
          if ($chars += word.chars + 1) > line-limit {
            take "\n" ~ word;
            $chars = word.chars;
          } else {
            take ' ' ~ word;
          }
        }
      }
    }
    return str;
  }

  method output-block {
    ($.output-format ~~ any Markdown, Gemtext) ?? "```\n" !! ''
  }
}

class Reporter does OutputHelper {
  has Hash %.aggregates is required;
  has UInt $.limit is required;
  has Category $.category = Host;
  has Metric $.metric is required;

  method report returns Str {
    join '', gather {
      with self!table -> (@table, %size) {
        my \format = '|' ~ join '|', " %{%size<count>}s ",
                     " %{%size<name>}s ", " %{%size<value>}s ", "\n";
        my \border = '+' ~ join '+',  '-' x (2+%size<count>),
                     '-' x (2+%size<name>), '-' x (2+%size<value>), "\n";
        take
          "{self.output-header}Top {$.limit} {$.metric}'s by {$.category}\n\n", 
          self.output-trim(%DESCRIPTION{$.metric}, border.chars), "\n\n",
          self.output-block, border,
          sprintf(format, 'Pos', $.category, $.metric),
          border;

        for @table -> \position, \name, \value {
          take sprintf format, position, name, value;
        }

        take border, self.output-block;
      }
    }
  }

  method !table returns List {
    my $count = 0;
    my @table;

    # Initial table size
    my %size =
      :count('Pos'.chars), :name($.category.chars),
      :value($.metric.chars);

    for self.sort-by($.metric) -> Aggregate \what {
      my \active = what.is-active ?? '*' !! ' ';
      my \name = active ~ what.name;
      my \value = self.human-str($.metric, what).Str;

      # Adjust size
      %size{.key} = .value if %size{.key} < .value
        for :count($count.Str.chars+1), :name(name.chars), :value(value.chars);

      @table.push: "{$count+1}.", name, value;
      last if ++$count == $.limit;
    }

    return @table, %size;
  }
  multi method sort-by(Uptime) { self.sort-by: *.uptime }
  multi method sort-by(Boots) { self.sort-by: *.boots }
  multi method sort-by(Score) { self.sort-by: *.meta-score }

  multi method sort-by(Code:D $sort-by) {
    %!aggregates{$!category}.values.sort(&$sort-by).reverse;
  }

  multi method human-str(Uptime, Aggregate:D $what) { Epoch.new($what.uptime).human-duration }
  multi method human-str(Boots, Aggregate:D $what) { $what.boots }
  multi method human-str(Score, Aggregate:D $what) { $what.meta-score }
}

class HostReporter is Reporter {
  multi method sort-by(Downtime) { self.sort-by: *.downtime }
  multi method sort-by(Lifespan) { self.sort-by: *.lifespan }

  multi method human-str(Downtime, Aggregate:D $what) { Epoch.new($what.downtime).human-duration }
  multi method human-str(Lifespan, Aggregate:D $what) { Epoch.new($what.lifespan).human-duration }
}

multi MAIN(
  Str :$stats-dir is required, #= The uptimed raw record input dir.
  Category :$category = Host, #= The category, one of Host, Kernel, KernelMajor, KernelName [default: 'Host']
  Metric :$metric = Uptime, #= The metric, one of Boots, Uptime, Score, Downtime, Lifespan
  UInt :$limit = 20, #= Limit output to num of entries.
  OutputFormat :$output-format = Plaintext, #= Output format.
) {
  my Hash %aggregates = Aggregator.new($stats-dir).aggregate;

  if $category ~~ Host {
    print HostReporter.new(:%aggregates, :$metric, :$limit, :$output-format).report;
  } elsif $metric ~~ MetricSubset {
    print Reporter.new(:%aggregates, :$category, :$metric, :$limit, :$output-format).report;
  } else {
    die "Category $category only supports the following metrics: {Metric.^enum_value_list.grep: * ~~ MetricSubset}";
  }
}

multi MAIN(
  Str :$stats-dir is required,
  Bool :$all, #= Generate all possible stats but Kernel (too verbose)
  Bool :$include-kernel, #= Also include Kernel
  UInt :$limit = 20,
  OutputFormat :$output-format = Plaintext,
) {
  my $header-indent = 2;
  my %aggregates = Aggregator.new($stats-dir).aggregate;

  for Category.^enum_value_list X Metric.^enum_value_list -> ($category, $metric) {
    next if !$include-kernel and $category ~~ Kernel;
    next if $category !~~ Host and $metric !~~ MetricSubset;
    if $category ~~ Host {
      print HostReporter.new(:%aggregates, :$metric, :$limit, :$output-format, :$header-indent).report
    } else {
      print Reporter.new(:%aggregates, :$category, :$metric, :$limit, :$output-format, :$header-indent).report;
    }
    say '';
  }
}

multi MAIN('test') {
  use Test;

  my @cross-product = gather {
    for Category.^enum_value_list X Metric.^enum_value_list X OutputFormat.^enum_value_list
    -> ($category, $metric, $output-format) {
      next if $category !~~ Host and $metric !~~ MetricSubset;
      take $category, $metric, $output-format;
    }
  }

  plan @cross-product;
  my $limit = 3;
  my %aggregates = Aggregator.new('./fixtures').aggregate;

  for @cross-product -> ($category, $metric, $output-format) {
    my \reporter = $category ~~ Host
      ?? HostReporter.new: :%aggregates, :$metric, :$limit, :$output-format
      !! Reporter.new: :%aggregates, :$category, :$metric, :$limit, :$output-format;
      #my $fh = open "./fixtures/$category.$metric.$output-format.expected", :w;
      #$fh.print(reporter.report);
      #$fh.close;
    is reporter.report, "./fixtures/$category.$metric.$output-format.expected".IO.slurp;
  }

  done-testing;
}
