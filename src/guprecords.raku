#!/usr/bin/env raku

use v6.d;

subset Cat of Str where * eq any <hostname os os-major uname>;
subset SubCat of Str where * eq any <boots uptime downtime lifespan metascore>;
subset PosInt of Int where * >= 0;

class Aggregate {
  has Str $.name is required;
  has PosInt $.uptime;
  has PosInt $.first-boot;
  has PosInt $.last-seen;
  has PosInt $.boots;

  method add-record(Str:D :$uptime is readonly, Str:D :$boot-time is readonly) {
      $!uptime += $uptime;
      $!first-boot = +$boot-time if not defined $!first-boot
                                 or $!first-boot > $boot-time;
      my Int $last-seen = $uptime + $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen
                               or $!last-seen < $last-seen;
      $!boots++;
  }

  method downtime returns PosInt {
    say qq:to/END/;
      name: {$.name}
      num boots: {$!boots}
      last seen: {$.last-seen}
      first boot: {$.first-boot}
      uptime: {$.uptime}
      result: {$.last-seen - $.first-boot - $.uptime}
    END
    $.last-seen - $.first-boot - $.uptime
  }
  method lifespan returns PosInt { self.downtime + $.uptime }

  method metascore returns PosInt {
    my \day = 1 * 24 * 3600;
    my \month = 30 * 24 * 3600;
    PosInt((($!uptime * 2) + self.downtime + ($!boots * day) + (self!is-active ?? month !! 0))/1000000)
  }

  method Str returns Str {
    qq:to/END/;
      {$!name}{self!is-active ?? ' (still active)' !! ''}
         uptime:     {duration($!uptime)}
         downtime:   {duration(self.downtime)}
         lifespan:   {duration(self.lifespan)}
         last seen:  {date($!last-seen)}
         first boot: {date($!first-boot)}
         num boots:  {$!boots}
         meta score: {self.metascore}
    END
  }

  method !is-active(PosInt:D \limit = 90) returns Bool {
    (DateTime.now - DateTime.new(Instant.from-posix: $!last-seen)) < limit * 3600 * 24;
  }

  sub duration(PosInt:D \seconds) returns Str {
    my DateTime \dt .= new(Instant.from-posix: seconds);
    return "{dt.year-1970} years, {dt.month} months, {dt.day} days";
  }

  sub date(PosInt:D \epoch) returns Str {
    DateTime.new(Instant.from-posix: epoch).yyyy-mm-dd
  }
}

class Aggregator {
  has Hash %.aggregates = { hostname => {}, os => {}, uname => {}, os-major => {} }

  method add-file(IO::Path:D :$file is readonly) {
    my Str $hostname = $file.IO.basename.split('.').first;
    %!aggregates<hostname>{$hostname} //= Aggregate.new: :name($hostname);
    for $file.IO.lines -> Str $line { self!add-line(:$line, :$hostname) }
  }

  method !add-line(Str:D :$line is readonly, Str:D :$hostname is readonly) {
    my Str ($uptime, $boot-time, $os) = $line.trim.split(':');
    my Str $uname = $os.split(' ').first;
    my Str $os-major = "$uname {$os.split(' ')[1].split('.').first}...";

    %!aggregates<os>{$os} //= Aggregate.new: :name($os);
    %!aggregates<uname>{$uname} //= Aggregate.new: :name($uname);
    %!aggregates<os-major>{$os-major} //= Aggregate.new: :name($os-major);

    for %!aggregates<hostname>{$hostname},
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
      $what.Str.say;
    }
  }

  multi method sort-by('uptime') { self.sort-by: *.uptime }
  multi method sort-by('downtime') { self.sort-by: *.downtime }
  multi method sort-by('lifespan') { self.sort-by: *.lifespan }
  multi method sort-by('boots') { self.sort-by: *.boots }
  multi method sort-by('metascore') { self.sort-by: *.metascore }

  multi method sort-by(Code:D $sort-by) {
    %!aggregates{$!cat}.values.sort(&$sort-by).reverse
  }
}

sub MAIN(
  Str :$stats-dir is required, #= The uptimed raw record input dir.
  Cat :$cat = 'hostname';      #= Category, one of hostname, os os-major and uname.
  SubCat :$sub-cat = 'uptime'; #= Sort by one of boots uptime downtime and lifespan.
  Str :$host-UNTESTED = '.*';           #= Hostname filter pattern.
) {

  my Aggregator $agg .= new;
  for dir($stats-dir, test => { /.records$/ }) -> $file {
    $file.say;
    $agg.add-file(:$file)
  }

  my Reporter $reporter .= new: :aggregates($agg.aggregates), :$cat, :$sub-cat;
  $reporter.report;
}
