#!/usr/bin/env raku

use v6.d;

subset Nat of Int where * >= 0;
subset Cat of Str where * eq any <host os os-major uname>;
subset SubCat of Str where * eq any <boots uptime meta-score downtime lifespan>;
subset HostOnlyCat of Cat where * eq 'host';
subset BasicSubCat of SubCat where * ne any <downtime lifespan>;

our Nat constant DAY = 1 * 24 * 3600;
our Nat constant MONTH = 30 * DAY;

class Epoch {
  has Nat $.value is required;

  submethod new (Nat $value) { self.bless(:$value) }

  method human-duration returns Str {
    my DateTime \dt .= new(Instant.from-posix: $!value);
    "{dt.year-1970} years, {dt.month} months, {dt.day} days";
  }

  method human-date returns Str {
    DateTime.new(Instant.from-posix: $!value).yyyy-mm-dd;
  }

  method newer-than(Nat:D \limit) returns Bool {
    (DateTime.now - DateTime.new(Instant.from-posix: $!value)) < limit * DAY;
  }
}

class Aggregate {
  has Str $.name is required;
  has Nat $.uptime;
  has Nat $.first-boot;
  has Nat $.last-seen;
  has Nat $.boots;

  method new (Str:D $name) { self.bless(:$name) }

  method add-record(Str:D :$uptime is readonly, Str:D :$boot-time is readonly) {
      my Int $last-seen = $uptime + $boot-time;
      $!uptime += $uptime;
      $!boots++;

      $!first-boot = +$boot-time if not defined $!first-boot or $!first-boot > $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen or $!last-seen < $last-seen;
  }

  method meta-score returns Nat {
    Nat((($!uptime * 2) + ($!boots * DAY) + (self.is-active ?? MONTH !! 0))/1000000)
  }

  method is-active(Nat:D \limit = 90) returns Bool {
    Epoch.new($!last-seen).newer-than: limit;
  }
}

class HostAggregate is Aggregate {
  method lifespan returns Nat { $.last-seen - $.first-boot }
  method downtime returns Nat { self.lifespan - $.uptime }
  method meta-score returns Nat { Nat(self.downtime / 1000000) + callsame }
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
  has Cat $.cat is required;
  has SubCat $.sub-cat is required;
  has SubCat $.sub-cat2 is required;
  has Nat $.first is required;
  has Hash %.aggregates;

  method report {
    say "Top {$.first} {$.sub-cat}'s by {$.cat}:\n";
    with self!table -> (@table, %size) {
      my Str \format = '|' ~ join '|',
        " %{%size<count>}s ", " %{%size<name>}s ",
        " %{%size<value>}s ", " %{%size<value2>}s ", "\n";
      my Str \border = '+' ~ join '+',  
        '-' x (2+%size<count>), '-' x (2+%size<name>),
        '-' x (2+%size<value>), '-' x (2+%size<value2>), "\n";
      print border;
      printf format, 'Pos', $.cat, $.sub-cat, $.sub-cat2;
      print border;
      for @table -> \position, \name, \value, \value2 {
        printf format, position, name, value, value2;
      }
      print border;
    }
  }

  method !table returns List {
    my Nat $count = 0;
    my @table;

    # Initial table size
    my %size =
      :count('Pos'.chars), :name($.cat.chars),
      :value($.sub-cat.chars), :value2($.sub-cat2.chars);

    for self.sort-by($.sub-cat) -> Aggregate \what {
      my Str \active = what.is-active ?? '*' !! ' ';
      my Str \name = active ~ what.name;
      my Str \value = self.human-str($.sub-cat, what).Str;
      my Str \value2 = self.human-str($.sub-cat2, what).Str;

      # Adjust size
      %size{.key} = .value if %size{.key} < .value for
        :count($count.Str.chars+1), :name(name.chars),
        :value(value.chars), :value2(value2.chars);

      @table.push: "{$count+1}.", name, value, value2;
      last if ++$count == $.first;
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
  Str :$stats-dir is required, #= The uptimed raw record input dir.
  HostOnlyCat :$cat = 'host',  #= Category, one of host, os os-major and uname.
  SubCat :$sub-cat = 'uptime', #= Sort by one of boots uptime downtime, lifespan and meta-score.
  SubCat :$sub-cat2 = 'uname', #= Additional sub-category to be displayed.
  Nat :$first = 20,            #= Only show top N entries.
) {
  do-it($stats-dir, HostReporter.new(:$cat, :$sub-cat, :$sub-cat2, :$first));
}

multi MAIN(
  Str :$stats-dir is required,
  Cat :$cat,
  BasicSubCat :$sub-cat = 'uptime',
  BasicSubCat :$sub-cat2 = 'boots',
  Nat :$first = 20,
) {
  do-it($stats-dir, Reporter.new(:$cat, :$sub-cat, :$sub-cat2, :$first));
}

