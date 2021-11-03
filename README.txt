NAME
    guprecords - Global uptime records

    Shows uprecord stats of several hosts at once.

  Synopsis
    guprecords [--help] [--total|--all] [--count=i] [--indir=s]

  Parameters
    --all
        Shows every individual uptime of all hosts.

    --count=i
        Show i num of entries. Default is 23.

    --nofqdn
        Don't display the FQDNs

    --help
        Shows the help

    --indir=s
        Read all the *.records files from dir s. Default is ./

    --total
        Aggregates a total uptime for every single host.

  Quick getting started
   Uptimed
    Firstival, you need to collect uprecords using the uptimed deaemon. To
    install it run:

        sudo aptitude install uptimed

    Please consult the uptimed and uprecords manpages. Please ensure to
    understand how it works and what it does.

    uptimed collects uprecords to

      /var/spool/uptimed/records

    And this file is used by guprecords for further processing.

   Collect all the uprecords
    You may have several hosts with uptimed running already. Collect all the
    records file to a central repository (e.g. git). Name each file
    FQDN.records and run

      guprecords --indir ./

    You may automate the collecting of all the uprecords using something
    like cron or puppet.

LICENSE
    See package description or project website.

AUTHOR
    Paul Buetow - <http://guprecords.buetow.org>

