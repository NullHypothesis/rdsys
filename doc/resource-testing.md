Resource testing
================

Rdsys tests its resources and only hands out resources that are known to work.
The actual testing is done by a separate service,
[bridgestrap](https://gitlab.torproject.org/tpo/anti-censorship/bridgestrap).
Rdsys requests resource tests by talking to bridgestrap's
[HTTP API](https://gitlab.torproject.org/tpo/anti-censorship/bridgestrap#input)
(the API endpoint is set in rdsys's
[configuration file](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/blob/9859ddda143eb5109b01be8ffcb76b683d37d819/conf/config.json#L4)).
When a resource is first added to rdsys, it is in state "untested".  Once it's
tested, it's either in state "functional" or "dysfunctional".

Mechanism
---------

When rdsys first learns about a new resource, it adds the resource to a
[testing pool](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/blob/9859ddda143eb5109b01be8ffcb76b683d37d819/internal/bridgestrap.go#L45).
This testing pool is sent to bridgestrap after it reaches its capacity of
25 resources, or one minute has passed – whatever happens first.

Resources are re-tested after they expire, i.e. once their
[expiry timer](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/blob/9859ddda143eb5109b01be8ffcb76b683d37d819/pkg/core/domain.go#L42)
exceeds the time they were last tested.  For Tor bridges, this happens after
[18 hours](https://gitlab.torproject.org/tpo/anti-censorship/rdsys/-/blob/9859ddda143eb5109b01be8ffcb76b683d37d819/pkg/usecases/resources/transports.go#L51).

Note that bridgestrap implements a test cache, so resources are not tested each
time they are sent to bridgestrap.  By default, bridgestrap caches a resource's
test result for 18 hours – identical to the expiry time of Tor bridges.  Rdsys
does not maintain a cache, so every test request is sent directly to
bridgestrap.  This isn't a problem because all communication happens over the
loopback interface.

Resource status page
--------------------

Rdsys exposes a Web page that allows bridge operators to query their bridge's
status.  The page is available at:

    https://bridges.torproject.org/status?id=FINGERPRINT

The page takes as input a bridge's fingerprint and shows all of the given
bridge's pluggable transports and their respective status (untested, functional,
or dysfunctional).

When a Tor bridge is first set up,
[it logs a URL](https://gitlab.torproject.org/tpo/core/tor/-/issues/30477)
to the above status page, allowing its operator to easily check its status.
