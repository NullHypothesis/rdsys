rdsys
=====

Rdsys is a distribution system for circumvention proxies and related resources.

This project is a partial reimplementation of [BridgeDB](https://gitlab.torproject.org/tpo/anti-censorship/bridgedb).

It is meant to be more flexible and robust in production environments and even potentially language-agnostic, as the distributors operate independently of the backend.

Currently, rdsys is in experimental, conceptual stages and it is not being used in any production environments.

## Installation

### Backend

#### Building

In order to build the backend, run:

	`go build -o ./rdsys-backend ./cmd/backend/main.go`

#### Installing

To install the binary, copy `./rdsys-backend` to a location in your PATH. (Eg: `/usr/local/bin`)

### Distributors

You can build and use multiple different distributors with rdsys. The following instructions involve building the HTTPS distributor.

#### Building

In order to build the HTTPS distributor, run:

	`go build -o ./rdsys-https ./cmd/distributors/https/main.go`

#### Installing

To install, copy `./rdsys-https` to a location in your PATH. (Eg: `/usr/local/bin`)

