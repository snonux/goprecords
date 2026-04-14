# mage goprecords a microservice

* Should run as a daemon, e.g. `goprecords --daemon`
* Should have a configurable database path to a local directory
* Local database is simply the input dir of the current goprecords CLI tool
* Introduce an API to generate the available output formats, also include a HTML output format. All should be configurable using a query-parameter to the API.
* Introduce an API to let clients upload new uprecords. see example database at /home/paul/git/uprecords/stats so every client should be able to upload 5 different files, HOSTNAME.{txt,cur.txt,records,os.txt,cpuinfo.txt}.
* Use token-based authentication (e.g. API keys or JWT) for upload endpoints.
* Upload should simply work using a curl CLI command given the auth is provided in the HTTP header.
* have a way to manage different auth keys per client. not via API but via a local database and a `goprecords --create-client-key HOSTNAME`.




