# guprecords - Global uptime records

`guprecords` is a Raku-based command-line program that generates uptime reports for hosts based on the input record files from `uptimed`. It allows you to create reports for different categories and metrics, and supports multiple output formats.

## Features

- Supports multiple categories: `Host`, `Kernel`, `KernelMajor`, and `KernelName`
- Supports multiple metrics: `Boots`, `Uptime`, `Score`, `Downtime`, and `Lifespan`
- Output formats available: `Plaintext`, `Markdown`, and `Gemtext`
- Provides top entries based on the specified limit

Whereas:

* `Boots`: Total count of system boots.
* `Uptime`: Total hosts uptimes.
* `Score`: A meta score, calculated combining all other metrics.
* `Downtime`: Total hosts downtimes.
* `Lifespan`: Downtimes plus uptimes.

## Usage

The program can be invoked with various command-line options to customize the generated reports. The following are the command-line options available:

- `--stats-dir`: The path to the directory containing the uptimed raw record input files (required)
- `--category`: The category to generate the report for, one of `Host`, `Kernel`, `KernelMajor`, `KernelName` (default: `Host`)
- `--metric`: The metric to use for the report, one of `Boots`, `Uptime`, `Score`, `Downtime`, `Lifespan` (default: `Uptime`)
- `--limit`: Limit the output to the specified number of entries (default: 20)
- `--output-format`: The output format for the report, one of `Plaintext`, `Markdown`, `Gemtext` (default: `Plaintext`)
- `--all`: Generate all possible reports except `Kernel` (optional)
- `--include-kernel`: Include the `Kernel` category when generating all reports (optional)
- `--stats-order`: Comma-separated list of `Category:Metric` pairs to order sections for `--all` (optional). Unlisted sections are appended in the default order.

### Example Usage

```bash
./guprecords.raku --stats-dir="./records" --category=Host --metric=Uptime --limit=10 --output-format=Markdown
```

This command generates a Markdown-formatted report for the top 10 hosts with the highest uptime.

```bash
./guprecords.raku --stats-dir="./records" --all --stats-order="Host:Uptime,Host:Boots"
```

This command generates all reports, placing the `Host:Uptime` section first, followed by `Host:Boots`, and then the remaining sections in the default order.

## Classes

- `Epoch`: A class representing the epoch value.
- `Aggregate`: A class representing the aggregate data for a specific category.
- `HostAggregate`: A subclass of `Aggregate` for handling host-related data.
- `Aggregator`: A class responsible for aggregating data from the record files.
- `OutputHelper`: A role providing helper methods for report output formatting.
- `Reporter`: A class generating reports based on specified category, metric, limit, and output format.
- `HostReporter`: A subclass of `Reporter` for handling host-related data.

## Test

The program includes test functionality. To run the tests, invoke the program with the `test` argument:

```bash
./guprecords.raku test
```

This will run the tests and report the results.

## End-to-end usage

### Use `uptimed` to produce uptime statistics

First, you need to generate uptime statistics from all hosts by installing and running `uptimed`. There's a package available for most common Linux and *BSD distributions nowadays. It's also available for macOS (Darwin) via Homebrew. For example, under Fedora, run `sudo dnf install uptimed`. https://github.com/rpodgorny/uptimed

### Collect all uptime records to a central location

Second, you must collect the `records` files produced by `uptimed`. Which are the raw uptime statistic files continuously updated by `uptimed`. Depending on the operating system used, the location of the records file can vary. It is advisable to store all the record files in a central git repository. 

An example records file looks like this:

```
11175544:1658053426:OpenBSD 7.1
10033984:1669229566:OpenBSD 7.2
7701011:1642849465:OpenBSD 7.0
3900089:1650550947:OpenBSD 7.1
3573912:1654452258:OpenBSD 7.1
2132201:1640713822:OpenBSD 7.0
88762:1640625045:OpenBSD 7.0
18452:1640603646:OpenBSD 7.0
3408:1642846040:OpenBSD 7.0
2315:1640622113:OpenBSD 7.0
1190:1654451052:OpenBSD 7.1
334:1650550601:OpenBSD 7.1
310:1669229245:OpenBSD 7.2
310:1640624443:OpenBSD 7.0
261:1640624769:OpenBSD 7.0
144:1669229090:OpenBSD 7.2
```

... whereas the first number is the total uptime since boot, and the second is the boot time. The last column identifies the operating system and kernel version.

`guprecords` does not provide any out-of-the-box solution for the collection part. I use a quick-and-dirty `Makefile` in a `uptimes.git` repository, where I can run `make push` to manually collect and push the uptime statistics of the current host to the git repository. I log in to all my machines anyway, sooner or later, and I also automatically run a shell script (via my login RC file) to re-collect the stats when they weren't collected for a week or so. So this solution is "just good enough" for me for now:

```
manual:
    records_path=/var/spool/uptimed/records; \
    test -f /usr/local/var/uptimed/records && records_path=/usr/local/var/uptimed/records; \
    test -f /var/db/uptimed/records && records_path=/var/db/uptimed/records; \
    cp $$records_path ./stats/$$(hostname | cut -d. -f1).records
    uprecords -a -m 100 > ./stats/$$(hostname | cut -d. -f1).txt
    uprecords -a | grep '^->' > ./stats/$$(hostname | cut -d. -f1).cur.txt
    git add ./stats/*
    git commit -a -m 'new uptime
push: manual
    git add ./stats/$(hostname)*
    git commit -a -m 'new stats'
    git pull origin master
    git push origin master
```

### Generate global uptime stats

Third, now you can finally run:

```
raku guprecords.raku --stats=dir=$HOME/git/uprecords/stats --all
```

... to generate something like the following:

## Top 20 Boots's by Host

Boots is the total number of host boots over the entire lifespan.

```
+-----+----------------+-------+------------------------------+
| Pos |           Host | Boots |                  Last Kernel |
+-----+----------------+-------+------------------------------+
|  1. |  alphacentauri |   671 |      FreeBSD 11.4-RELEASE-p7 |
|  2. |           mars |   207 |          Linux 3.2.0-4-amd64 |
|  3. |         *earth |   182 | Linux 6.14.5-300.fc42.x86_64 |
|  4. |       callisto |   153 |  Linux 4.0.4-303.fc22.x86_64 |
|  5. |       dionysus |   136 |     FreeBSD 13.0-RELEASE-p11 |
|  6. |      tauceti-e |   120 |          Linux 3.2.0-4-amd64 |
|  7. |      *makemake |    76 |  Linux 6.9.9-200.fc40.x86_64 |
|  8. |        *uranus |    59 |                  NetBSD 10.1 |
|  9. |          pluto |    51 |          Linux 3.2.0-4-amd64 |
| 10. |      mega15289 |    50 |                Darwin 23.4.0 |
| 11. |    *fishfinger |    43 |                  OpenBSD 7.6 |
| 12. |          *t450 |    43 |         FreeBSD 14.2-RELEASE |
| 13. |   *mega-m3-pro |    41 |                Darwin 24.4.0 |
| 14. |         phobos |    40 |      Linux 3.4.0-CM-g1dd7cdf |
| 15. |       mega8477 |    40 |                Darwin 13.4.0 |
| 16. |      *blowfish |    38 |                  OpenBSD 7.6 |
| 17. |            sun |    33 |     FreeBSD 10.3-RELEASE-p24 |
| 18. |            *f2 |    25 |      FreeBSD 14.2-RELEASE-p1 |
| 19. |            *f1 |    20 |      FreeBSD 14.2-RELEASE-p1 |
| 20. |           moon |    20 |      FreeBSD 14.0-RELEASE-p3 |
+-----+----------------+-------+------------------------------+
```

## Top 20 Uptime's by Host

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+----------------+-----------------------------+-----------------------------------+
| Pos |           Host |                      Uptime |                       Last Kernel |
+-----+----------------+-----------------------------+-----------------------------------+
|  1. |         vulcan |   4 years, 5 months, 6 days | Linux 3.10.0-1160.81.1.el7.x86_64 |
|  2. |            sun |  3 years, 9 months, 26 days |          FreeBSD 10.3-RELEASE-p24 |
|  3. |        *uranus |   3 years, 9 months, 5 days |                       NetBSD 10.1 |
|  4. |      *blowfish |  3 years, 5 months, 16 days |                       OpenBSD 7.6 |
|  5. |         *earth |   3 years, 5 months, 6 days |      Linux 6.14.5-300.fc42.x86_64 |
|  6. |          uugrn |   3 years, 5 months, 5 days |           FreeBSD 11.2-RELEASE-p4 |
|  7. |      deltavega |  3 years, 1 months, 21 days | Linux 3.10.0-1160.11.1.el7.x86_64 |
|  8. |          pluto | 2 years, 10 months, 29 days |               Linux 3.2.0-4-amd64 |
|  9. |    *fishfinger |  2 years, 9 months, 11 days |                       OpenBSD 7.6 |
| 10. |        tauceti |  2 years, 3 months, 19 days |               Linux 3.2.0-4-amd64 |
| 11. |      mega15289 | 1 years, 12 months, 17 days |                     Darwin 23.4.0 |
| 12. |      tauceti-f |  1 years, 9 months, 18 days |               Linux 3.2.0-3-amd64 |
| 13. |          *t450 |  1 years, 4 months, 28 days |              FreeBSD 14.2-RELEASE |
| 14. |       mega8477 |  1 years, 3 months, 25 days |                     Darwin 13.4.0 |
| 15. |          host0 |   1 years, 3 months, 9 days |          FreeBSD 6.2-RELEASE-p5   |
| 16. |      *makemake |   1 years, 3 months, 5 days |       Linux 6.9.9-200.fc40.x86_64 |
| 17. |      tauceti-e |  1 years, 2 months, 20 days |               Linux 3.2.0-4-amd64 |
| 18. |   *mega-m3-pro | 0 years, 12 months, 13 days |                     Darwin 24.4.0 |
| 19. |       callisto | 0 years, 10 months, 31 days |       Linux 4.0.4-303.fc22.x86_64 |
| 20. |  alphacentauri | 0 years, 10 months, 28 days |           FreeBSD 11.4-RELEASE-p7 |
+-----+----------------+-----------------------------+-----------------------------------+
```

## Top 20 Score's by Host

Score is calculated by combining all other metrics.

```
+-----+----------------+-------+-----------------------------------+
| Pos |           Host | Score |                       Last Kernel |
+-----+----------------+-------+-----------------------------------+
|  1. |        *uranus |   342 |                       NetBSD 10.1 |
|  2. |         vulcan |   275 | Linux 3.10.0-1160.81.1.el7.x86_64 |
|  3. |            sun |   238 |          FreeBSD 10.3-RELEASE-p24 |
|  4. |         *earth |   236 |      Linux 6.14.5-300.fc42.x86_64 |
|  5. |      *blowfish |   218 |                       OpenBSD 7.6 |
|  6. |          uugrn |   211 |           FreeBSD 11.2-RELEASE-p4 |
|  7. |  alphacentauri |   201 |           FreeBSD 11.4-RELEASE-p7 |
|  8. |      deltavega |   193 | Linux 3.10.0-1160.11.1.el7.x86_64 |
|  9. |          pluto |   182 |               Linux 3.2.0-4-amd64 |
| 10. |    *fishfinger |   176 |                       OpenBSD 7.6 |
| 11. |       dionysus |   156 |          FreeBSD 13.0-RELEASE-p11 |
| 12. |      mega15289 |   147 |                     Darwin 23.4.0 |
| 13. |        tauceti |   141 |               Linux 3.2.0-4-amd64 |
| 14. |      *makemake |   131 |       Linux 6.9.9-200.fc40.x86_64 |
| 15. |      tauceti-f |   108 |               Linux 3.2.0-3-amd64 |
| 16. |          *t450 |   106 |              FreeBSD 14.2-RELEASE |
| 17. |      tauceti-e |    96 |               Linux 3.2.0-4-amd64 |
| 18. |       callisto |    86 |       Linux 4.0.4-303.fc22.x86_64 |
| 19. |       mega8477 |    80 |                     Darwin 13.4.0 |
| 20. |          host0 |    76 |          FreeBSD 6.2-RELEASE-p5   |
+-----+----------------+-------+-----------------------------------+
```

## Top 20 Downtime's by Host

Downtime is the total downtime of a host over the entire lifespan.

```
+-----+----------------+-----------------------------+------------------------------+
| Pos |           Host |                    Downtime |                  Last Kernel |
+-----+----------------+-----------------------------+------------------------------+
|  1. |       dionysus |  8 years, 3 months, 16 days |     FreeBSD 13.0-RELEASE-p11 |
|  2. |        *uranus |  6 years, 7 months, 31 days |                  NetBSD 10.1 |
|  3. |  alphacentauri | 5 years, 11 months, 18 days |      FreeBSD 11.4-RELEASE-p7 |
|  4. |      *makemake |   3 years, 2 months, 2 days |  Linux 6.9.9-200.fc40.x86_64 |
|  5. |           moon |   2 years, 1 months, 1 days |      FreeBSD 14.0-RELEASE-p3 |
|  6. |       callisto |  1 years, 5 months, 15 days |  Linux 4.0.4-303.fc22.x86_64 |
|  7. |      mega15289 |  1 years, 4 months, 24 days |                Darwin 23.4.0 |
|  8. |          *t450 |  1 years, 2 months, 13 days |         FreeBSD 14.2-RELEASE |
|  9. |           mars |  1 years, 2 months, 10 days |          Linux 3.2.0-4-amd64 |
| 10. |      tauceti-e |  0 years, 12 months, 9 days |          Linux 3.2.0-4-amd64 |
| 11. |         sirius |  0 years, 8 months, 20 days |   Linux 2.6.32-042stab111.12 |
| 12. |         *earth |  0 years, 6 months, 19 days | Linux 6.14.5-300.fc42.x86_64 |
| 13. |         deimos |  0 years, 5 months, 15 days |  Linux 4.4.5-300.fc23.x86_64 |
| 14. |            *f0 |  0 years, 4 months, 20 days |      FreeBSD 14.2-RELEASE-p1 |
| 15. |            *f2 |  0 years, 4 months, 19 days |      FreeBSD 14.2-RELEASE-p1 |
| 16. |            *f1 |  0 years, 4 months, 18 days |      FreeBSD 14.2-RELEASE-p1 |
| 17. |        joghurt |   0 years, 2 months, 9 days |    FreeBSD 7.0-PRERELEASE    |
| 18. |          host0 |   0 years, 2 months, 1 days |     FreeBSD 6.2-RELEASE-p5   |
| 19. |      fibonacci |  0 years, 1 months, 11 days |     FreeBSD 5.3-RELEASE-p15  |
| 20. |          cobol |   0 years, 1 months, 8 days |     FreeBSD 10.1-RELEASE-p24 |
+-----+----------------+-----------------------------+------------------------------+
```

## Top 20 Lifespan's by Host

Lifespan is the total uptime + the total downtime of a host.

```
+-----+----------------+-----------------------------+-----------------------------------+
| Pos |           Host |                    Lifespan |                       Last Kernel |
+-----+----------------+-----------------------------+-----------------------------------+
|  1. |        *uranus |  10 years, 4 months, 5 days |                       NetBSD 10.1 |
|  2. |       dionysus |  8 years, 6 months, 17 days |          FreeBSD 13.0-RELEASE-p11 |
|  3. |  alphacentauri |  6 years, 9 months, 13 days |           FreeBSD 11.4-RELEASE-p7 |
|  4. |         vulcan |   4 years, 5 months, 6 days | Linux 3.10.0-1160.81.1.el7.x86_64 |
|  5. |      *makemake |   4 years, 4 months, 7 days |       Linux 6.9.9-200.fc40.x86_64 |
|  6. |         *earth | 3 years, 10 months, 23 days |      Linux 6.14.5-300.fc42.x86_64 |
|  7. |            sun |  3 years, 10 months, 2 days |          FreeBSD 10.3-RELEASE-p24 |
|  8. |      *blowfish |  3 years, 5 months, 17 days |                       OpenBSD 7.6 |
|  9. |          uugrn |   3 years, 5 months, 5 days |           FreeBSD 11.2-RELEASE-p4 |
| 10. |      mega15289 |   3 years, 4 months, 9 days |                     Darwin 23.4.0 |
| 11. |      deltavega |  3 years, 1 months, 21 days | Linux 3.10.0-1160.11.1.el7.x86_64 |
| 12. |          pluto | 2 years, 10 months, 30 days |               Linux 3.2.0-4-amd64 |
| 13. |    *fishfinger |  2 years, 9 months, 13 days |                       OpenBSD 7.6 |
| 14. |          *t450 |   2 years, 6 months, 9 days |              FreeBSD 14.2-RELEASE |
| 15. |           moon |  2 years, 4 months, 25 days |           FreeBSD 14.0-RELEASE-p3 |
| 16. |        tauceti |  2 years, 3 months, 22 days |               Linux 3.2.0-4-amd64 |
| 17. |       callisto |  2 years, 3 months, 13 days |       Linux 4.0.4-303.fc22.x86_64 |
| 18. |      tauceti-e |  2 years, 1 months, 29 days |               Linux 3.2.0-4-amd64 |
| 19. |      tauceti-f |  1 years, 9 months, 20 days |               Linux 3.2.0-3-amd64 |
| 20. |           mars |  1 years, 8 months, 19 days |               Linux 3.2.0-4-amd64 |
+-----+----------------+-----------------------------+-----------------------------------+
```

## Top 20 Boots's by KernelMajor

Boots is the total number of host boots over the entire lifespan.

```
+-----+----------------+-------+
| Pos |    KernelMajor | Boots |
+-----+----------------+-------+
|  1. |  FreeBSD 10... |   551 |
|  2. |     Linux 3... |   550 |
|  3. |     Linux 5... |   162 |
|  4. |    *Linux 6... |   162 |
|  5. |     Linux 4... |   161 |
|  6. |  FreeBSD 11... |   153 |
|  7. |  FreeBSD 13... |   116 |
|  8. |  *OpenBSD 7... |    91 |
|  9. | *FreeBSD 14... |    79 |
| 10. |   Darwin 13... |    40 |
| 11. |   Darwin 23... |    33 |
| 12. |   FreeBSD 5... |    25 |
| 13. |     Linux 2... |    22 |
| 14. |   Darwin 21... |    17 |
| 15. |   Darwin 15... |    15 |
| 16. |  *Darwin 24... |    13 |
| 17. |   Darwin 22... |    12 |
| 18. |   Darwin 18... |    11 |
| 19. |   OpenBSD 4... |    10 |
| 20. |   FreeBSD 6... |    10 |
+-----+----------------+-------+
```

## Top 20 Uptime's by KernelMajor

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+----------------+------------------------------+
| Pos |    KernelMajor |                       Uptime |
+-----+----------------+------------------------------+
|  1. |     Linux 3... | 15 years, 10 months, 25 days |
|  2. |  *OpenBSD 7... |   6 years, 9 months, 24 days |
|  3. |  FreeBSD 10... |    5 years, 9 months, 9 days |
|  4. |     Linux 5... |  4 years, 10 months, 21 days |
|  5. |    *Linux 6... |    2 years, 8 months, 3 days |
|  6. |     Linux 4... |   2 years, 7 months, 22 days |
|  7. |  FreeBSD 11... |   2 years, 4 months, 28 days |
|  8. |     Linux 2... |  1 years, 11 months, 21 days |
|  9. | *FreeBSD 14... |    1 years, 6 months, 1 days |
| 10. |   Darwin 13... |   1 years, 3 months, 25 days |
| 11. |   FreeBSD 6... |    1 years, 3 months, 9 days |
| 12. |   Darwin 23... |   0 years, 11 months, 9 days |
| 13. |   OpenBSD 4... |   0 years, 8 months, 12 days |
| 14. |   Darwin 21... |    0 years, 8 months, 2 days |
| 15. |   Darwin 18... |    0 years, 7 months, 5 days |
| 16. |   Darwin 22... |   0 years, 6 months, 22 days |
| 17. |   Darwin 15... |   0 years, 6 months, 15 days |
| 18. |   FreeBSD 5... |   0 years, 5 months, 18 days |
| 19. |  *Darwin 24... |   0 years, 4 months, 16 days |
| 20. |  FreeBSD 13... |    0 years, 4 months, 2 days |
+-----+----------------+------------------------------+
```

## Top 20 Score's by KernelMajor

Score is calculated by combining all other metrics.

```
+-----+----------------+-------+
| Pos |    KernelMajor | Score |
+-----+----------------+-------+
|  1. |     Linux 3... |  1045 |
|  2. |  *OpenBSD 7... |   435 |
|  3. |  FreeBSD 10... |   406 |
|  4. |     Linux 5... |   317 |
|  5. |    *Linux 6... |   179 |
|  6. |     Linux 4... |   175 |
|  7. |  FreeBSD 11... |   159 |
|  8. |     Linux 2... |   121 |
|  9. | *FreeBSD 14... |    98 |
| 10. |   Darwin 13... |    80 |
| 11. |   FreeBSD 6... |    75 |
| 12. |   Darwin 23... |    56 |
| 13. |   OpenBSD 4... |    39 |
| 14. |   Darwin 21... |    38 |
| 15. |   Darwin 18... |    32 |
| 16. |   Darwin 22... |    30 |
| 17. |   Darwin 15... |    29 |
| 18. |  FreeBSD 13... |    25 |
| 19. |   FreeBSD 5... |    25 |
| 20. |  *Darwin 24... |    21 |
+-----+----------------+-------+
```

## Top 20 Boots's by KernelName

Boots is the total number of host boots over the entire lifespan.

```
+-----+------------+-------+
| Pos | KernelName | Boots |
+-----+------------+-------+
|  1. |     *Linux |  1057 |
|  2. |   *FreeBSD |   944 |
|  3. |    *Darwin |   146 |
|  4. |   *OpenBSD |   101 |
|  5. |    *NetBSD |     1 |
+-----+------------+-------+
```

## Top 20 Uptime's by KernelName

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+------------+-----------------------------+
| Pos | KernelName |                      Uptime |
+-----+------------+-----------------------------+
|  1. |     *Linux | 27 years, 8 months, 25 days |
|  2. |   *FreeBSD |  11 years, 5 months, 3 days |
|  3. |   *OpenBSD |   7 years, 5 months, 5 days |
|  4. |    *Darwin |   4 years, 8 months, 4 days |
|  5. |    *NetBSD |   0 years, 1 months, 1 days |
+-----+------------+-----------------------------+
```

## Top 20 Score's by KernelName

Score is calculated by combining all other metrics.

```
+-----+------------+-------+
| Pos | KernelName | Score |
+-----+------------+-------+
|  1. |     *Linux |  1839 |
|  2. |   *FreeBSD |   799 |
|  3. |   *OpenBSD |   474 |
|  4. |    *Darwin |   304 |
|  5. |    *NetBSD |     2 |
+-----+------------+-------+
```
