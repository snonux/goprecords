#!/usr/bin/env raku

use v6.d;

subset Nat of Int where * >= 0;
subset Cat of Str where * eq any <host os os-major uname>;
subset SubCat of Str where * eq any <boots uptime downtime lifespan meta-score>;

class Epoch {
  our Nat constant DAY = 1 * 24 * 3600;
  our Nat constant MONTH = 30 * DAY;
  has Nat $.value is required;

  submethod new (Nat $value) { self.bless(:$value) }

  method duration returns Str {
    my DateTime \dt .= new(Instant.from-posix: $!value);
    "{dt.year-1970} years, {dt.month} months, {dt.day} days";
  }

  method date returns Str {
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

  method add-record(Str:D :$uptime is readonly, Str:D :$boot-time is readonly) {
      my Int $last-seen = $uptime + $boot-time;
      $!uptime += $uptime;
      $!boots++;

      $!first-boot = +$boot-time if not defined $!first-boot or $!first-boot > $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen or $!last-seen < $last-seen;
  }

  method meta-score returns Nat {
    Nat((($!uptime * 2) + ($!boots * Epoch.DAY) + (self.is-active ?? Epoch.MONTH !! 0))/1000000)
  }

  method is-active(Nat:D \limit = 90) returns Bool {
    Epoch.new($!last-seen).newer-than: limit;
  }
}

class HostAggregate is Aggregate {
  method lifespan returns Nat { $.last-seen - $.first-boot }
  method downtime returns Nat { self.lifespan - $.uptime }
  method meta-score returns Nat { self.downtime + callsame }
}

class Aggregator {
  has Hash %.aggregates = { host => {}, os => {}, uname => {}, os-major => {} }

  method add-file(IO::Path:D :$file is readonly) {
    my Str $host = $file.IO.basename.split('.').first;

    die "Record file for {$host} already processed - duplicate inputs?"
      if %!aggregates<host>{$host}:exists;
    %!aggregates<host>{$host} = HostAggregate.new: :name($host);

    for $file.IO.lines -> Str $line { self!add-line(:$line, :$host) }
  }

  method !add-line(Str:D :$line is readonly, Str:D :$host is readonly) {
    my Str ($uptime, $boot-time, $os) = $line.trim.split(':');
    my Str $uname = $os.split(' ').first;
    my Str $os-major = "$uname {$os.split(' ')[1].split('.').first}...";

    %!aggregates<os>{$os} //= Aggregate.new: :name($os);
    %!aggregates<uname>{$uname} //= Aggregate.new: :name($uname);
    %!aggregates<os-major>{$os-major} //= Aggregate.new: :name($os-major);

    for %!aggregates<host>{$host},
        %!aggregates<os>{$os},
        %!aggregates<uname>{$uname},
        %!aggregates<os-major>{$os-major} {
      .add-record(:$uptime, :$boot-time);
    }
  }
}

class Reporter {
  has Hash %.aggregates is required;
  has Cat $.cat is required;
  has SubCat $.sub-cat is required;

  method report {
    for self.sort-by($!sub-cat) -> $what {
      $what.raku.say;
      $what.is-active.say;
    }
  }

  multi method sort-by('uptime') { self.sort-by: *.uptime }
  multi method sort-by('downtime') { self.sort-by: *.downtime }
  multi method sort-by('lifespan') { self.sort-by: *.lifespan }
  multi method sort-by('boots') { self.sort-by: *.boots }
  multi method sort-by('meta-score') { self.sort-by: *.meta-score }

  multi method sort-by(Code:D $sort-by) {
    %!aggregates{$!cat}.values.sort(&$sort-by).reverse
  }
}

multi MAIN(
  Str :$stats-dir is required, #= The uptimed raw record input dir.
  Cat :$cat = 'host';          #= Category, one of host, os os-major and uname.
  SubCat :$sub-cat = 'uptime'; #= Sort by one of boots uptime downtime and lifespan.
  Str :$host-UNTESTED = '.*';  #= Hostname filter pattern.
) {
  my Aggregator $agg .= new;
  for dir($stats-dir, test => { /.records$/ }) -> $file {
    $agg.add-file(:$file)
  }

  my Reporter $reporter .= new: :aggregates($agg.aggregates), :$cat, :$sub-cat;
  $reporter.report;
}
