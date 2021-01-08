Rdsys
=====

**Disclaimer: This project is under construction.  Nothing is stable.**

Rdsys is short for *r*esource *d*istribution *sys*tem: Resources related to
censorship circumvention (e.g. proxies or download links) are handed out by a
variety of distribution methods to censored users.  The goal is to supply
censored users with circumvention proxies, allowing these users to join us on
the open Internet.

Rdsys supersedes
[BridgeDB](https://gitlab.torproject.org/tpo/anti-censorship/bridgedb).
Functionality-wise, rdsys and BridgeDB overlap but rdsys is neither a subset
nor a superset of BridgeDB's functionality.

Installation
============

1. Compile the backend executable:

        go build -o rdsys-backend cmd/backend/main.go

1. Compile the distributors executable:

        go build -o rdsys-distributor cmd/distributors/main.go

Usage
=====

1. Start the backend by running:

        ./rdsys-backend -config /path/to/config.json

2. Start a distributor, which distributes the backend's resources:

        ./rdsys-distributor -name salmon -config /path/to/config.json

More documentation
==================

* [Design and architecture](doc/architecture.md)
* [Resource testing](doc/resource-testing.md)
