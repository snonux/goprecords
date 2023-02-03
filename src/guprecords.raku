#!/usr/bin/env raku

use v6.d;

class Aggregate {
  has Int $.total-uptime = 0;
  has Int $.first-boot;
  has Int $.last-seen;

  method aggregate(Str :$uptime is readonly, Str :$boot-time is readonly) {
      $!total-uptime += $uptime;
      $!first-boot = +$boot-time if not defined $!first-boot
                                 or $!first-boot > +$boot-time;
      my Int $last-seen = $uptime + $boot-time;
      $!last-seen = $last-seen if not defined $!last-seen
                               or $!last-seen < $last-seen;
  }

  method total-downtime { $.last-seen - $.first-boot - $.total-uptime }
  method total-time { self.total-downtime + $.total-uptime }
}

class Aggregator {
  has %.aggregates = { hostname => {}, os => {}, os-kernel => {} }

  method aggregate(IO::Path :$file is readonly) {
    my Str $hostname = $file.IO.basename.split('.').first;
    %!aggregates<hostname>{$hostname} //= Aggregate.new;

    for $file.IO.lines {
      my Str ($uptime, $boot-time, $os) = .trim.split(':');
      my Str $os-kernel = $os.split(' ').first;

      %!aggregates<os>{$os} //= Aggregate.new;
      %!aggregates<os-kernel>{$os-kernel} //= Aggregate.new;

      for %!aggregates.values.flatmap( { .values } ) {
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
    for $aggregates.kv -> $name, $aggregate {
      say "\t$name $aggregate";
    }
  }
}
