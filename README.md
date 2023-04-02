# guprecords - Global uptime records

`guprecords` is a program developed in Raku which combines and analysis a bunch of uptime records collected by `uptimed` from multiple hosts. `uptimed` can only analyse and display stats from one single host, whereas `guprecords` combines the stats (records files) of many hosts.

## Usage

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
+-----+-----------------+-------+
| Pos |            Host | Boots |
+-----+-----------------+-------+
|  1. |   alphacentauri |   671 |
|  2. |            mars |   207 |
|  3. |        callisto |   153 |
|  4. |          uranus |   150 |
|  5. |        dionysus |   136 |
|  6. |       tauceti-e |   120 |
|  7. |          *earth |   108 |
|  8. |           pluto |    51 |
|  9. | *ultramega15289 |    50 |
| 10. |        makemake |    50 |
| 11. |   ultramega8477 |    40 |
| 12. |          phobos |    40 |
| 13. |             sun |    33 |
| 14. |           *t450 |    25 |
| 15. |         *vulcan |    19 |
| 16. |         tauceti |    16 |
| 17. |       *blowfish |    16 |
| 18. |     sagittarius |    15 |
| 19. |       deltavega |    12 |
| 20. |         twofish |    10 |
+-----+-----------------+-------+
```

## Top 20 Uptime's by Host

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+-----------------+-----------------------------+
| Pos |            Host |                      Uptime |
+-----+-----------------+-----------------------------+
|  1. |         *vulcan |   4 years, 4 months, 7 days |
|  2. |          uranus | 3 years, 11 months, 21 days |
|  3. |             sun |  3 years, 9 months, 26 days |
|  4. |           uugrn |   3 years, 5 months, 5 days |
|  5. |       deltavega |  3 years, 1 months, 21 days |
|  6. |           pluto | 2 years, 10 months, 29 days |
|  7. |         tauceti |  2 years, 3 months, 19 days |
|  8. |       tauceti-f |  1 years, 9 months, 18 days |
|  9. | *ultramega15289 |  1 years, 7 months, 29 days |
| 10. |          *earth |  1 years, 4 months, 24 days |
| 11. |   ultramega8477 |  1 years, 3 months, 25 days |
| 12. |       *blowfish |  1 years, 3 months, 24 days |
| 13. |           host0 |   1 years, 3 months, 9 days |
| 14. |       tauceti-e |  1 years, 2 months, 20 days |
| 15. |        makemake |   1 years, 1 months, 6 days |
| 16. |        callisto | 0 years, 10 months, 31 days |
| 17. |   alphacentauri | 0 years, 10 months, 28 days |
| 18. |          london |  0 years, 9 months, 16 days |
| 19. |         twofish |  0 years, 8 months, 31 days |
| 20. |        fishbone |  0 years, 8 months, 12 days |
+-----+-----------------+-----------------------------+
```

## Top 20 Score's by Host

Score is calculated by combining all other metrics.

```
+-----+-----------------+-------+
| Pos |            Host | Score |
+-----+-----------------+-------+
|  1. |          uranus |   360 |
|  2. |   alphacentauri |   294 |
|  3. |        dionysus |   285 |
|  4. |         *vulcan |   273 |
|  5. |             sun |   238 |
|  6. |           uugrn |   211 |
|  7. |       deltavega |   193 |
|  8. |           pluto |   182 |
|  9. |         tauceti |   141 |
| 10. | *ultramega15289 |   122 |
| 11. |       tauceti-e |   111 |
| 12. |       tauceti-f |   108 |
| 13. |        callisto |   108 |
| 14. |          *earth |   105 |
| 15. |        makemake |    96 |
| 16. |            mars |    85 |
| 17. |       *blowfish |    81 |
| 18. |   ultramega8477 |    80 |
| 19. |           host0 |    77 |
| 20. |          sirius |    52 |
+-----+-----------------+-------+
```

## Top 20 Downtime's by Host

Downtime is the total downtime of a host over the entire lifespan.

```
+-----+-----------------+-----------------------------+
| Pos |            Host |                    Downtime |
+-----+-----------------+-----------------------------+
|  1. |        dionysus |  8 years, 3 months, 16 days |
|  2. |   alphacentauri | 5 years, 11 months, 18 days |
|  3. |          uranus |  3 years, 3 months, 28 days |
|  4. |        callisto |  1 years, 5 months, 15 days |
|  5. |            mars |  1 years, 2 months, 10 days |
|  6. |       tauceti-e |  0 years, 12 months, 9 days |
|  7. |        makemake | 0 years, 11 months, 22 days |
|  8. |          sirius |  0 years, 8 months, 20 days |
|  9. | *ultramega15289 |   0 years, 7 months, 8 days |
| 10. |          deimos |  0 years, 5 months, 15 days |
| 11. |          *earth |  0 years, 5 months, 10 days |
| 12. |           *t450 |  0 years, 2 months, 23 days |
| 13. |         joghurt |   0 years, 2 months, 9 days |
| 14. |           host0 |   0 years, 2 months, 1 days |
| 15. |       fibonacci |  0 years, 1 months, 11 days |
| 16. |           cobol |   0 years, 1 months, 8 days |
| 17. |   ultramega8477 |   0 years, 1 months, 8 days |
| 18. |             sun |   0 years, 1 months, 7 days |
| 19. |          sentax |   0 years, 1 months, 6 days |
| 20. |     sagittarius |   0 years, 1 months, 6 days |
+-----+-----------------+-----------------------------+
```

## Top 20 Lifespan's by Host

Lifespan is the total uptime + the total downtime of a host.

```
+-----+-----------------+-----------------------------+
| Pos |            Host |                    Lifespan |
+-----+-----------------+-----------------------------+
|  1. |        dionysus |  8 years, 6 months, 17 days |
|  2. |          uranus |  7 years, 2 months, 16 days |
|  3. |   alphacentauri |  6 years, 9 months, 13 days |
|  4. |         *vulcan |   4 years, 4 months, 7 days |
|  5. |             sun |  3 years, 10 months, 2 days |
|  6. |           uugrn |   3 years, 5 months, 5 days |
|  7. |       deltavega |  3 years, 1 months, 21 days |
|  8. |           pluto | 2 years, 10 months, 30 days |
|  9. |         tauceti |  2 years, 3 months, 22 days |
| 10. |        callisto |  2 years, 3 months, 13 days |
| 11. | *ultramega15289 |   2 years, 2 months, 3 days |
| 12. |       tauceti-e |  2 years, 1 months, 29 days |
| 13. |        makemake | 1 years, 11 months, 28 days |
| 14. |       tauceti-f |  1 years, 9 months, 20 days |
| 15. |          *earth |  1 years, 8 months, 31 days |
| 16. |            mars |  1 years, 8 months, 19 days |
| 17. |           host0 |  1 years, 4 months, 10 days |
| 18. |   ultramega8477 |   1 years, 4 months, 1 days |
| 19. |       *blowfish |  1 years, 3 months, 24 days |
| 20. |          sirius |  1 years, 2 months, 24 days |
+-----+-----------------+-----------------------------+
```

## Top 20 Boots's by Kernel

Boots is the total number of host boots over the entire lifespan.

```
+-----+------------------------------+-------+
| Pos |                       Kernel | Boots |
+-----+------------------------------+-------+
|  1. |          Linux 3.2.0-4-amd64 |   452 |
|  2. |  Linux 4.0.4-303.fc22.x86_64 |   103 |
|  3. |         FreeBSD 10.1-RELEASE |    76 |
|  4. |         FreeBSD 10.0-RELEASE |    52 |
|  5. |     FreeBSD 10.1-RELEASE-p19 |    48 |
|  6. |     FreeBSD 11.1-RELEASE-p10 |    46 |
|  7. |                Darwin 13.4.0 |    40 |
|  8. |     FreeBSD 10.3-RELEASE-p11 |    39 |
|  9. |     FreeBSD 10.3-RELEASE-p20 |    39 |
| 10. |     FreeBSD 10.1-RELEASE-p10 |    38 |
| 11. |      FreeBSD 10.0-RELEASE-p7 |    38 |
| 12. |     FreeBSD 10.1-RELEASE-p35 |    34 |
| 13. |      FreeBSD 13.0-RELEASE-p4 |    29 |
| 14. |      FreeBSD 10.0-RELEASE-p9 |    29 |
| 15. |     FreeBSD 10.3-RELEASE-p24 |    28 |
| 16. |         FreeBSD 13.0-RELEASE |    26 |
| 17. |     *FreeBSD 13.1-RELEASE-p3 |    25 |
| 18. |     FreeBSD 10.1-RELEASE-p31 |    24 |
| 19. |      FreeBSD 11.4-RELEASE-p5 |    23 |
| 20. |     FreeBSD 10.1-RELEASE-p41 |    20 |
+-----+------------------------------+-------+
```

## Top 20 Uptime's by Kernel

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+------------------------------------+-----------------------------+
| Pos |                             Kernel |                      Uptime |
+-----+------------------------------------+-----------------------------+
|  1. |                Linux 3.2.0-4-amd64 |  6 years, 4 months, 20 days |
|  2. | *Linux 3.10.0-1160.15.2.el7.x86_64 | 1 years, 12 months, 15 days |
|  3. |                Linux 3.2.0-3-amd64 |  1 years, 9 months, 18 days |
|  4. |   Linux 3.10.0-957.21.3.el7.x86_64 |  1 years, 7 months, 13 days |
|  5. |           FreeBSD 10.1-RELEASE-p16 |  1 years, 5 months, 23 days |
|  6. |                      Darwin 13.4.0 |  1 years, 3 months, 25 days |
|  7. |               Linux 2.6.32-5-amd64 |  1 years, 3 months, 18 days |
|  8. |               FreeBSD 10.2-RELEASE |  1 years, 3 months, 11 days |
|  9. |                        OpenBSD 7.1 |  1 years, 2 months, 28 days |
| 10. |  Linux 3.10.0-1160.11.1.el7.x86_64 | 0 years, 12 months, 21 days |
| 11. |            FreeBSD 11.0-RELEASE-p1 |  0 years, 10 months, 9 days |
| 12. |        Linux 4.0.4-303.fc22.x86_64 |  0 years, 9 months, 17 days |
| 13. |                FreeBSD 6.2-RELEASE |  0 years, 9 months, 14 days |
| 14. |  Linux 3.10.0-1127.13.1.el7.x86_64 |   0 years, 9 months, 9 days |
| 15. |                       *OpenBSD 7.2 |   0 years, 9 months, 3 days |
| 16. |            FreeBSD 11.1-RELEASE-p4 |  0 years, 8 months, 18 days |
| 17. |                        OpenBSD 4.1 |  0 years, 8 months, 12 days |
| 18. |    Linux 3.10.0-957.1.3.el7.x86_64 |  0 years, 8 months, 12 days |
| 19. |                        OpenBSD 7.0 |  0 years, 8 months, 10 days |
| 20. |                      Darwin 18.7.0 |  0 years, 7 months, 30 days |
+-----+------------------------------------+-----------------------------+
```

## Top 20 Score's by Kernel

Score is calculated by combining all other metrics.

```
+-----+------------------------------------+-------+
| Pos |                             Kernel | Score |
+-----+------------------------------------+-------+
|  1. |                Linux 3.2.0-4-amd64 |   436 |
|  2. | *Linux 3.10.0-1160.15.2.el7.x86_64 |   126 |
|  3. |                Linux 3.2.0-3-amd64 |   108 |
|  4. |   Linux 3.10.0-957.21.3.el7.x86_64 |    96 |
|  5. |           FreeBSD 10.1-RELEASE-p16 |    88 |
|  6. |                      Darwin 13.4.0 |    80 |
|  7. |               Linux 2.6.32-5-amd64 |    77 |
|  8. |               FreeBSD 10.2-RELEASE |    75 |
|  9. |                        OpenBSD 7.1 |    73 |
| 10. |  Linux 3.10.0-1160.11.1.el7.x86_64 |    61 |
| 11. |        Linux 4.0.4-303.fc22.x86_64 |    53 |
| 12. |            FreeBSD 11.0-RELEASE-p1 |    48 |
| 13. |                       *OpenBSD 7.2 |    45 |
| 14. |                FreeBSD 6.2-RELEASE |    44 |
| 15. |  Linux 3.10.0-1127.13.1.el7.x86_64 |    43 |
| 16. |            FreeBSD 11.1-RELEASE-p4 |    39 |
| 17. |                        OpenBSD 7.0 |    39 |
| 18. |                        OpenBSD 4.1 |    39 |
| 19. |    Linux 3.10.0-957.1.3.el7.x86_64 |    38 |
| 20. |                      Darwin 18.7.0 |    37 |
+-----+------------------------------------+-------+
```

## Top 20 Boots's by KernelMajor

Boots is the total number of host boots over the entire lifespan.

```
+-----+----------------+-------+
| Pos |    KernelMajor | Boots |
+-----+----------------+-------+
|  1. |  FreeBSD 10... |   551 |
|  2. |    *Linux 3... |   550 |
|  3. |     Linux 5... |   249 |
|  4. |     Linux 4... |   164 |
|  5. |  FreeBSD 11... |   153 |
|  6. | *FreeBSD 13... |   123 |
|  7. |   Darwin 13... |    40 |
|  8. |  *OpenBSD 7... |    34 |
|  9. |    *Linux 6... |    29 |
| 10. |   FreeBSD 5... |    25 |
| 11. |     Linux 2... |    22 |
| 12. |  *Darwin 21... |    20 |
| 13. |   Darwin 15... |    15 |
| 14. |   Darwin 18... |    15 |
| 15. |   Darwin 20... |    12 |
| 16. |   FreeBSD 7... |    10 |
| 17. |   FreeBSD 6... |    10 |
| 18. |   OpenBSD 4... |    10 |
| 19. |  *Darwin 22... |     2 |
| 20. |   Darwin 19... |     1 |
+-----+----------------+-------+
```

## Top 20 Uptime's by KernelMajor

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+----------------+-----------------------------+
| Pos |    KernelMajor |                      Uptime |
+-----+----------------+-----------------------------+
|  1. |    *Linux 3... | 15 years, 9 months, 26 days |
|  2. |  FreeBSD 10... |   5 years, 9 months, 9 days |
|  3. |     Linux 5... |  3 years, 12 months, 2 days |
|  4. |     Linux 4... |   2 years, 8 months, 9 days |
|  5. |  *OpenBSD 7... |   2 years, 6 months, 9 days |
|  6. |  FreeBSD 11... |  2 years, 4 months, 28 days |
|  7. |     Linux 2... | 1 years, 11 months, 21 days |
|  8. |   Darwin 13... |  1 years, 3 months, 25 days |
|  9. |   FreeBSD 6... |   1 years, 3 months, 9 days |
| 10. |  *Darwin 21... |  0 years, 8 months, 20 days |
| 11. |   OpenBSD 4... |  0 years, 8 months, 12 days |
| 12. |   Darwin 18... |  0 years, 7 months, 30 days |
| 13. |   Darwin 15... |  0 years, 6 months, 15 days |
| 14. |   FreeBSD 5... |  0 years, 5 months, 18 days |
| 15. |    *Linux 6... |  0 years, 5 months, 17 days |
| 16. |   Darwin 20... |  0 years, 4 months, 22 days |
| 17. | *FreeBSD 13... |  0 years, 3 months, 29 days |
| 18. |   FreeBSD 7... |   0 years, 2 months, 5 days |
| 19. |  *Darwin 22... |  0 years, 1 months, 14 days |
| 20. |   Darwin 19... |   0 years, 1 months, 9 days |
+-----+----------------+-----------------------------+
```

## Top 20 Score's by KernelMajor

Score is calculated by combining all other metrics.

```
+-----+----------------+-------+
| Pos |    KernelMajor | Score |
+-----+----------------+-------+
|  1. |    *Linux 3... |  1043 |
|  2. |  FreeBSD 10... |   406 |
|  3. |     Linux 5... |   268 |
|  4. |     Linux 4... |   178 |
|  5. |  *OpenBSD 7... |   159 |
|  6. |  FreeBSD 11... |   159 |
|  7. |     Linux 2... |   121 |
|  8. |   Darwin 13... |    80 |
|  9. |   FreeBSD 6... |    75 |
| 10. |  *Darwin 21... |    44 |
| 11. |   OpenBSD 4... |    39 |
| 12. |   Darwin 18... |    37 |
| 13. |   Darwin 15... |    29 |
| 14. | *FreeBSD 13... |    28 |
| 15. |    *Linux 6... |    28 |
| 16. |   FreeBSD 5... |    25 |
| 17. |   Darwin 20... |    20 |
| 18. |   FreeBSD 7... |     7 |
| 19. |  *Darwin 22... |     5 |
| 20. |   Darwin 19... |     1 |
+-----+----------------+-------+
```

## Top 20 Boots's by KernelName

Boots is the total number of host boots over the entire lifespan.

```
+-----+------------+-------+
| Pos | KernelName | Boots |
+-----+------------+-------+
|  1. |     *Linux |  1014 |
|  2. |   *FreeBSD |   872 |
|  3. |    *Darwin |   105 |
|  4. |   *OpenBSD |    44 |
+-----+------------+-------+
```

## Top 20 Uptime's by KernelName

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+------------+-----------------------------+
| Pos | KernelName |                      Uptime |
+-----+------------+-----------------------------+
|  1. |     *Linux | 24 years, 7 months, 10 days |
|  2. |   *FreeBSD | 9 years, 11 months, 29 days |
|  3. |    *Darwin |   3 years, 4 months, 3 days |
|  4. |   *OpenBSD |  3 years, 1 months, 18 days |
+-----+------------+-----------------------------+
```

## Top 20 Score's by KernelName

Score is calculated by combining all other metrics.

```
+-----+------------+-------+
| Pos | KernelName | Score |
+-----+------------+-------+
|  1. |     *Linux |  1637 |
|  2. |   *FreeBSD |   703 |
|  3. |    *Darwin |   217 |
|  4. |   *OpenBSD |   198 |
+-----+------------+-------+
```

