#!/usr/bin/env raku

use v6.d;

subset Natural of Int where * >= 0;
subset Category of Str where * eq any <host os os-major uname>;
subset HostMetric of Str where * eq any <boots uptime meta-score downtime lifespan>;
subset Metric of HostMetric where * ne any <downtime lifespan>;

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
  has Hash %.aggregates = { host => {}, os => {}, uname => {}, os-major => {} }

  method add-file(IO::Path:D $file is readonly) {
    my Str $host = $file.IO.basename.split('.').first;

    die "Record file for {$host} already processed - duplicate inputs?"
      if %!aggregates<host>{$host}:exists;
    %!aggregates<host>{$host} = HostAggregate.new($host);

    for $file.IO.lines -> Str $line { self!add-line(:$line, :$host) }
  }

  method !add-line(Str:D :$line is readonly, Str:D :$host is readonly) {
    my Str ($uptime, $boot-time, $os) = $line.trim.split(':');
    my Str $uname = $os.split(' ').first;
    my Str $os-major = "$uname {$os.split(' ')[1].split('.').first}...";

    %!aggregates<os>{$os} //= Aggregate.new($os);
    %!aggregates<uname>{$uname} //= Aggregate.new($uname);
    %!aggregates<os-major>{$os-major} //= Aggregate.new($os-major);

    for %!aggregates<host>{$host}, %!aggregates<os>{$os},
        %!aggregates<uname>{$uname}, %!aggregates<os-major>{$os-major} {
      .add-record(:$uptime, :$boot-time);
    }
  }
}

class Reporter {
  has Category $.cat is required;
  has Metric $.metric is required;
  has Natural $.limit is required;
  has Hash %.aggregates;

  method report {
    say "Top {$.limit} {$.metric}'s by {$.cat}:\n";
    with self!table -> (@table, %size) {
      my Str \format = '|' ~ join '|',
        " %{%size<count>}s ", " %{%size<name>}s ", " %{%size<value>}s ", "\n";
      my Str \border = '+' ~ join '+',  
        '-' x (2+%size<count>), '-' x (2+%size<name>), '-' x (2+%size<value>), "\n";
      print border;
      printf format, 'Pos', $.cat, $.metric;
      print border;
      for @table -> \position, \name, \value {
        printf format, position, name, value;
      }
      print border;
    }
  }

  method !table returns List {
    my Natural $count = 0;
    my @table;

    # Initial table size
    my %size =
      :count('Pos'.chars), :name($.cat.chars),
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

  multi method sort-by('uptime') { self.sort-by: *.uptime }
  multi method sort-by('boots') { self.sort-by: *.boots }
  multi method sort-by('meta-score') { self.sort-by: *.meta-score }

  multi method sort-by(Code:D $sort-by) {
    %!aggregates{$!cat}.values.sort(&$sort-by).reverse;
  }

  multi method human-str('uptime', Aggregate:D $what) { Epoch.new($what.uptime).human-duration }
  multi method human-str('boots', Aggregate:D $what) { $what.boots }
  multi method human-str('meta-score', Aggregate:D $what) { $what.meta-score }
}

class HostReporter is Reporter {
  multi method sort-by('downtime') { self.sort-by: *.downtime }
  multi method sort-by('lifespan') { self.sort-by: *.lifespan }

  multi method human-str('downtime', Aggregate:D $what) { Epoch.new($what.downtime).human-duration }
  multi method human-str('lifespan', Aggregate:D $what) { Epoch.new($what.lifespan).human-duration }
}

sub do-it(Str:D \stats-dir, Reporter:D \reporter) {
  my Aggregator \aggregator .= new;
  aggregator.add-file($_) for dir(stats-dir, test => { /.records$/ });
  reporter.aggregates = aggregator.aggregates;
  reporter.report;
}

multi MAIN(
  Str :$stats-dir is required,
  HostMetric :$metric = 'uptime',
  Natural :$limit = 20,
) {
  do-it($stats-dir, HostReporter.new(cat => 'host', :$metric, :$limit));
}

multi MAIN(
  Str :$stats-dir is required, #= The uptimed raw record input dir.
  Category :$cat is required where * ne 'host', #= The category, one of host, os, os-major, uname [default: 'host']
  Metric :$metric = 'uptime', #= The metric, one of boots, uptime, meta-score, downtime, lifespan
  Natural :$limit = 20, #= Limit output to num of entries.
) {
  do-it($stats-dir, HostReporter.new(:$cat, :$metric, :$limit));
}
