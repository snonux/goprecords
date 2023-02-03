#!/usr/bin/env raku

use v6.d;

class Aggregate {
  has Int $.total-uptime = 0;
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
) {
  my Aggregator $aggregator .= new;

  for dir($in-dir, test => { /\.records$/ }) -> $file {
    $aggregator.aggregate(:$file)
  }

  for $aggregator.aggregates.kv -> $category, $aggregates {
    say "Categoty $category";
    for $aggregates.kv -> $name, $agg {
      my Str $plural = $agg.elems > 1 ?? 's' !! '';
      say "\t$name (" ~ $agg.elems ~ " record$plural)";
    }
  }
}
