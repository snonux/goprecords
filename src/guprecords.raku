#!/usr/bin/env raku

use v6.d;

#= The stats major category.
subset Cat of Str where * eq any <hostname os os-major uname>;
#= The sub-cateogory.
subset SubCat of Str where * eq any <boots uptime downtime lifespan metascore>;

class Aggregate {
  has Str $.name is required;
  has Int $.uptime;
  has Int $.first-boot;
  has Int $.last-seen;
  has Int $.boots;

  method add-record(Str:D :$uptime is readonly, Str:D :$boot-time is readonly) {
      $!uptime += $uptime;
      $!first-boot = +$boot-time if not defined $!first-boot
                                 or $!first-boot > $boot-time;
      my Int $last-seen = $uptime + $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen
                               or $!last-seen < $last-seen;
      $!boots++;
  }

  method downtime returns Int { $.last-seen - $.first-boot - $.uptime }
  method lifespan returns Int { self.downtime + $.uptime }

  method metascore returns Int {
    my \week = 7 * 24 * 3600;
    my \month = 30 * 24 * 3600;
    Int((($!uptime * 2) + self.downtime + ($!boots * week) + (self!is-active ?? month !! 0))/1000000)
  }

  method Str returns Str {
    qq:to/END/;
      {$!name}{self!is-active ?? ' (still active)' !! ''}
         meta score: {self.metascore}
         uptime:     {duration($!uptime)}
         downtime:   {duration(self.downtime)}
         lifespan:   {duration(self.lifespan)}
         num boots:  {$!boots}
         first boot: {date($!first-boot)}
         last seen:  {date($!last-seen)}
    END
  }

  method !is-active(Int:D \limit = 90) returns Bool {
    (DateTime.now - DateTime.new(Instant.from-posix: $!last-seen)) < limit * 3600 * 24;
  }

  sub duration(Int:D \seconds) returns Str {
    my DateTime \dt .= new(Instant.from-posix: seconds);
    return "{dt.year-1970} years, {dt.month} months, {dt.day} days";
  }

  sub date(Int:D \epoch) returns Str {
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
) {
  my Aggregator $agg .= new;

  for dir($stats-dir, test => { /\.records$/ }) -> $file {
    $agg.add-file(:$file)
  }

  my Reporter $reporter .= new: :aggregates($agg.aggregates), :$cat, :$sub-cat;
  $reporter.report;
}
