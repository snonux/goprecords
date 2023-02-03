#!/usr/bin/env raku

use v6.d;

constant SECOND = 1;
constant MINUTE = 60 * SECOND;
constant HOUR = 60 * MINUTE;
constant DAY = 24 * HOUR;
constant MONTH = 24 * HOUR;

class Aggregate {
  has Int $.total-uptime;
  has Int $.first-boot;
  has Int $.last-seen;
  has Int $.elems;

  method aggregate(Str :$uptime is readonly, Str :$boot-time is readonly) {
      $!total-uptime += $uptime;
      $!first-boot = +$boot-time if not defined $!first-boot
                                 or $!first-boot > $boot-time;
      my Int $last-seen = $uptime + $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen
                               or $!last-seen < $last-seen;
      $!elems++;
  }

  method total-downtime { $.last-seen - $.first-boot - $.total-uptime }
  method total-time { self.total-downtime + $.total-uptime }

  method Str returns Str {
    #duration($!total-uptime)
    my Str $active = self.is-active ?? '* ' !! '  ';
    "$active {duration(self.total-downtime)} {date($!last-seen)}";
  }

  method is-active returns Bool {
    # Last seen 3 months ago?
    my Int \seconds = (DateTime.now - DateTime.new(Instant.from-posix: $!last-seen)).Int;
    seconds < 3600 * 24 * 90;
  }

  sub duration(Int \seconds) returns Str {
    my DateTime \dt .= new(Instant.from-posix: seconds);
    "{dt.year-1970} years, {dt.month} months, {dt.day} days";
  }

  sub date(Int \epoch) returns Str {
    DateTime.new(Instant.from-posix: epoch).yyyy-mm-dd
  }
}

class Aggregator {
  has %.aggregates = { hostname => {}, os => {}, uname => {} }

  method aggregate(IO::Path :$file is readonly) {
    my Str $hostname = $file.IO.basename.split('.').first;
    %!aggregates<hostname>{$hostname} //= Aggregate.new;

    for $file.IO.lines {
      my Str ($uptime, $boot-time, $os) = .trim.split(':');
      my Str $uname = $os.split(' ').first;

      %!aggregates<os>{$os} //= Aggregate.new;
      %!aggregates<uname>{$uname} //= Aggregate.new;

      for %!aggregates<hostname>{$hostname},
          %!aggregates<os>{$os},
          %!aggregates<uname>{$uname} {
        .aggregate(:$uptime, :$boot-time);
      }
    }
  }
}

sub MAIN(
  Str $in-dir = './stats',
  Str $sort-by = 'uptime';
) {
  my Aggregator $aggregator .= new;

  for dir($in-dir, test => { /\.records$/ }) -> $file {
    $aggregator.aggregate(:$file)
  }

  for $aggregator.aggregates.kv -> $category, $aggregates {
    say "Categoty $category";
    for $aggregates.kv -> $name, $agg {
      my Str $plural = $agg.elems > 1 ?? 's' !! '';
      say "\t$name $agg (" ~ $agg.elems ~ " record$plural)";
    }
  }
}
