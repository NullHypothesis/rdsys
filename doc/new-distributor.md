Implementing new distributors
=============================

This document explains the process of building a new rdsys distributor.  In a
nutshell, a new distributor requires the following code:

1. [Backend code](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/blob/master/pkg/usecases/distributors/dummy/dummy.go)
   that handles the distribution logic and the interaction with the backend.

2. [Frontend code](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/blob/master/pkg/presentation/distributors/dummy/web.go)
   that hands out resources to users.

3. Simple configuration and command line code.

Note that it's convenient to implement new distributors in Go (because one can
re-use existing code like the IPC API) but it's not necessary.  Distributors are
stand-alone processes that can be implemented in any language.

Conceptually, a distributor has backend and frontend code.  Backend code (part
of rdsys's
[usecases](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/tree/master/pkg/usecases)
layer) takes care of distribution logic and frontend code (part of rdsys's
[presentation](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/tree/master/pkg/presentation)
layer) takes care of handing resources out to users.  Some distributors, like
Salmon, have complex backend code but simple frontend code.  Other distributors,
like HTTPS, have simple backend code but more complex frontend code.

The separation between backend and frontend code brings with it the flexibility
to build multiple ways to access a distributor.  For example, one could write
different frontends for Salmon: In addition to the domain-fronted API, one could
build a command line interface or an SMTP-based interface.  The backend code
remains the same but the means via which users access the backend code differs.
