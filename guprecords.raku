#!/usr/bin/env raku

use v6.d;

enum Category <Host OS OSMajor Uname>;
enum Metric <Boots Uptime MetaScore Downtime Lifespan>;

enum OutputFormat <Plaintext Markdown Gemtext>;

subset MetricSubset of Metric where * ne any (Downtime, Lifespan);
subset Natural of Int where * >= 0;

our Natural constant DAY = 1 * 24 * 3600;
our Natural constant MONTH = 30 * DAY;

class Epoch {
  has Natural $.value is required;

  submethod new (Natural $value) { self.bless(:$value) }

  method human-duration returns Str {
    my DateTime \dt .= new(Instant.from-posix: $!value);
    "{dt.year-1970} years, {dt.month} months, {dt.day} days";
  }

  method human-date returns Str {
    DateTime.new(Instant.from-posix: $!value).yyyy-mm-dd;
  }

  method newer-than(Natural:D \limit) returns Bool {
    (DateTime.now - DateTime.new(Instant.from-posix: $!value)) < limit * DAY;
  }
}

class Aggregate {
  has Str $.name is required;
  has Natural $.uptime;
  has Natural $.first-boot;
  has Natural $.last-seen;
  has Natural $.boots;

  method new (Str:D $name) { self.bless(:$name) }

  method add-record(Str:D :$uptime is readonly, Str:D :$boot-time is readonly) {
      my Int $last-seen = $uptime + $boot-time;
      $!uptime += $uptime;
      $!boots++;

      $!first-boot = +$boot-time if not defined $!first-boot or $!first-boot > $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen or $!last-seen < $last-seen;
  }

  method meta-score returns Natural {
    Natural((($!uptime * 2) + ($!boots * DAY) + (self.is-active ?? MONTH !! 0))/1000000)
  }

  method is-active(Natural:D \limit = 90) returns Bool {
    Epoch.new($!last-seen).newer-than: limit;
  }
}

class HostAggregate is Aggregate {
  method lifespan returns Natural { $.last-seen - $.first-boot }
  method downtime returns Natural { self.lifespan - $.uptime }
  method meta-score returns Natural { Natural(self.downtime / 1000000) + callsame }
}

class Aggregator {
  has Hash %!aggregates = { Host => {}, OS => {}, Uname => {}, OSMajor => {} }
  has Str $.stats-dir is required;

  submethod new (Str:D $stats-dir) { self.bless(:$stats-dir) }

  method aggregate () {
    self!add-file($_) for dir($!stats-dir, test => { /.records$/ });
    return %!aggregates;
  }

  method !add-file(IO::Path:D $file is readonly) {
    my Str $host = $file.IO.basename.split('.').first;

    die "Record file for {$host} already processed - duplicate inputs?"
      if %!aggregates<Host>{$host}:exists;
    %!aggregates<Host>{$host} = HostAggregate.new($host);

    for $file.IO.lines -> Str $line { self!add-line(:$line, :$host) }
  }

  method !add-line(Str:D :$line is readonly, Str:D :$host is readonly) {
    my Str ($uptime, $boot-time, $os) = $line.trim.split(':');
    my Str $uname = $os.split(' ').first;
    my Str $os-major = "$uname {$os.split(' ')[1].split('.').first}...";

    %!aggregates<OS>{$os} //= Aggregate.new($os);
    %!aggregates<Uname>{$uname} //= Aggregate.new($uname);
    %!aggregates<OSMajor>{$os-major} //= Aggregate.new($os-major);

    for %!aggregates<Host>{$host}, %!aggregates<OS>{$os},
        %!aggregates<Uname>{$uname}, %!aggregates<OSMajor>{$os-major} {
      .add-record(:$uptime, :$boot-time);
    }
  }
}

class Reporter {
  has Hash %.aggregates is required;
  has OutputFormat $.output-format is required;
  has Natural $.limit is required;
  has Natural $.header-indent = 1;
  has Category $.category = Host;
  has Metric $.metric is required;

  method report {
    say "{self!output-header}Top {$.limit} {$.metric}'s by {$.category}:\n";

    with self!table -> (@table, %size) {
      my Str \format = '|' ~ join '|',
        " %{%size<count>}s ", " %{%size<name>}s ", " %{%size<value>}s ", "\n";
      my Str \border = '+' ~ join '+',  
        '-' x (2+%size<count>), '-' x (2+%size<name>), '-' x (2+%size<value>), "\n";

      say self!output-block;
      print border;
      printf format, 'Pos', $.category, $.metric;
      print border;

      for @table -> \position, \name, \value {
        printf format, position, name, value;
      }

      print border;
      say self!output-block;
    }
  }

  method !table returns List {
    my Natural $count = 0;
    my @table;

    # Initial table size
    my %size =
      :count('Pos'.chars), :name($.category.chars),
      :value($.metric.chars);

    for self.sort-by($.metric) -> Aggregate \what {
      my Str \active = what.is-active ?? '*' !! ' ';
      my Str \name = active ~ what.name;
      my Str \value = self.human-str($.metric, what).Str;

      # Adjust size
      %size{.key} = .value if %size{.key} < .value for
        :count($count.Str.chars+1), :name(name.chars), :value(value.chars);

      @table.push: "{$count+1}.", name, value;
      last if ++$count == $.limit;
    }

    return @table, %size;
  }

  method !output-header {
    ($.output-format ~~ any (Markdown, Gemtext)) ?? '#' x $.header-indent ~ ' ' !! ''
  }

  method !output-block {
    ($.output-format ~~ any (Markdown, Gemtext)) ?? '```' !! ''
  }

  multi method sort-by(Uptime) { self.sort-by: *.uptime }
  multi method sort-by(Boots) { self.sort-by: *.boots }
  multi method sort-by(MetaScore) { self.sort-by: *.meta-score }

  multi method sort-by(Code:D $sort-by) {
    %!aggregates{$!category}.values.sort(&$sort-by).reverse;
  }

  multi method human-str(Uptime, Aggregate:D $what) { Epoch.new($what.uptime).human-duration }
  multi method human-str(Boots, Aggregate:D $what) { $what.boots }
  multi method human-str(MetaScore, Aggregate:D $what) { $what.meta-score }
}

class HostReporter is Reporter {
  multi method sort-by(Downtime) { self.sort-by: *.downtime }
  multi method sort-by(Lifespan) { self.sort-by: *.lifespan }

  multi method human-str(Downtime, Aggregate:D $what) { Epoch.new($what.downtime).human-duration }
  multi method human-str(Lifespan, Aggregate:D $what) { Epoch.new($what.lifespan).human-duration }
}

multi sub MAIN(
  Str :$stats-dir is required, #= The uptimed raw record input dir.
  Category :$category = Host, #= The category, one of Host, OS, OSMajor, Uname [default: 'Host']
  Metric :$metric = Uptime, #= The metric, one of Boots, Uptime, MetaScore, Downtime, Lifespan
  Natural :$limit = 20, #= Limit output to num of entries.
  OutputFormat :$output-format = Plaintext, #= Output format.
) {
  my Hash %aggregates = Aggregator.new($stats-dir).aggregate;

  if $category ~~ Host {
    HostReporter.new(:%aggregates, :$metric, :$limit, :$output-format).report;
  } elsif $metric ~~ MetricSubset {
    Reporter.new(:%aggregates, :$category, :$metric, :$limit, :$output-format).report;
  } else {
    die "Category $category only supports the following metrics: {Metric.^enum_value_list.grep: * ~~ MetricSubset}";
  }
}

multi sub MAIN(
  Str :$stats-dir is required,
  Bool :$all, #= Generate all possible stats
  Natural :$limit = 20,
  OutputFormat :$output-format = Plaintext,
) {
  my Natural $header-indent = 2;
  my Hash %aggregates = Aggregator.new($stats-dir).aggregate;

  for Category.^enum_value_list X Metric.^enum_value_list -> (Category $category, Metric $metric) {
    next if $category !~~ Host and $metric !~~ MetricSubset;
    if $category ~~ Host {
      HostReporter.new(:%aggregates, :$metric, :$limit, :$output-format, :$header-indent).report
    } else {
      Reporter.new(:%aggregates, :$category, :$metric, :$limit, :$output-format, :$header-indent).report;
    }
    say '';
  }
}

multi sub MAIN('test') {
  use Test;
  plan 1;

  #do-it('./fixtures', 

  done-testing;
}
