# NekoQ-Bootstrap

NekoQ-Bootstrap is the core and fundamental service for NekoQ related services.

It is responsible for the bootstrap service discovery and basic configuration storage.

In order to make NekoQ-Bootstrap as simple as possible, the usecases are kept as essential ones.

## 1. Usage

1. run `go build` in folder `cmd/nekoq-bootstrap`
2. copy `bootstrap.toml.example` to `bootstrap.toml`
3. put them in the same folder
4. run `nekoq-bootstrap`

## 2. Architecture

```text
                         DC1                                                    DC2
 ┌──────────────────────────────────────────────────┐   ┌──────────────────────────────────────────────────┐
 │                                                  │   │                                                  │
 │ ┌─────────────────┐         ┌─────────────────┐  │   │  ┌─────────────────┐         ┌─────────────────┐ │
 │ │                 │         │                 ├──┼───┼─►│                 │         │                 │ │
 │ │ NekoQ Bootstrap │◄────────┤ NekoQ Discovery │  │   │  │ NekoQ Discovery ├────────►│ NekoQ Bootstrap │ │
 │ │     Cluster     │         │                 │◄─┼───┼──┤                 │         │     Cluster     │ │
 │ └─────────────────┘◄┐     ┌►└─────────────────┘  │   │  └─────────────────┘◄┐     ┌►└─────────────────┘ │
 │                     │     │                      │   │                      │     │                     │
 │                     │     │                      │   │                      │     │                     │
 │                     │     │                      │   │                      │     │                     │
 │                     │     │                      │   │                      │     │                     │
 │                     │     │                      │   │                      │     │                     │
 │               ┌─────┴─────┴─────┐                │   │                ┌─────┴─────┴─────┐               │
 │               │                 │                │   │                │                 │               │
 │               │ NekoQ Services  │                │   │                │ NekoQ Services  │               │
 │               │                 │                │   │                │                 │               │
 │               └─────────────────┘                │   │                └─────────────────┘               │
 │                                                  │   │                                                  │
 │                                                  │   │                                                  │
 │                                                  │   │                                                  │
 └──────────────────────────────────────────────────┘   └──────────────────────────────────────────────────┘

```

NekoQ-Boostrap is the bootstrap for all nekoq related and user services to discover fundamental services.

It acts as a service discovery but only do basic static and dynamic discovery within one single datacenter.

These services can be registered in nekoq-bootstrap:

* nekoq discovery service(embedded in nekoq)
* NekoQ-Security
* nekoq consistency system(embedded in nekoq)
* And simple storage

The workflow for a service to start is:

1. Query nekoq-bootstrap -> discovery + nekoq-security + consistency system + key configurations
2. Prepare authentications using nekoq-security
3. Find essential services using discovery
4. Get essential configurations from consistency system
5. Do initialization within the service
6. Ready to serve

The design principles:

1. Keep NekoQ-Bootstrap as simple as possible in order to get high availability
2. Only for several key components
3. Easy to configure & start
4. Keep as little as possible data to persist

Shared component types should be:

1. discovery
2. security
3. consistency
4. batch
5. message queue
6. agent/service bus

## 3. Feature List

### DNS module

* [X] DNS service discovery: A record
* [X] DNS service discovery: AAAA record
* [X] DNS service discovery: TXT record
* [X] DNS service discovery: SRV record
* [X] DNS upstream name server support
* [x] Default handler for unsupported dns query type - current NXDomain handler
* [ ] DNS prefix specified upstream name server support
* [X] DNS over http - rfc8484
* [ ] DNS over https - rfc8484
* [ ] DNS service discovery - AAAA/MX/CNAME(exclusive from A/AAAA)/Multiple A or AAAA for load balancing
* [ ] DNS service discovery - SOA/PTR
* [ ] DNS record load balancing via multiple records support - A/AAAA/SRV/etc.
* [ ] Wildcard DNS record
* [x] Enclosure DNS domain - domains with specific suffixes will not be leaked to upstream
* [x] Enclosure DNS domain supports: A/AAAA/TXT/SRV/PTR
* [ ] DNS Sec
* [ ] DNS TCP
* [ ] Recursive DNS
* [ ] Authority DNS Server
* [ ] Environment specified dns records, e.g. internal access records or external access records
* [x] DNS caching
* [ ] DNS TTL(both managed records and upstream responses)
* [ ] DNS recursion support
* [ ] DNS Authoritative
* [x] DNS resolve tracing log
* [ ] Standardize
* [x] DNS record dynamic loading
    * Via nekoq-component/configure and onlyconfig
    * Default location: selectors=`app=nekoq-bootstrap,dc=default,env=PROD`, group=`nekoq-bootstrap.dns`, key=`records`
    * Format: toml

### Http module

* [X] Register several types of service
    * Support register same node to several NekoQ-Bootstrap. Note: DO NOT use different data in this case. Otherwise
      only the latest registration will be effect under current HA strategy within the cluster.
* [X] Peer auth

### High available cluster module

* [X] Peer data sync
* [X] Peer auth
* [ ] Peer data sync: dns data

### Simple KV store module

* [ ] KV store

### Management

* [x] Web manager via nekoq-component/configure and onlyconfig
    * Standard configure client in nekoq-component/configure
    * Support onlyconfig server
    * Dynamic configure from webmgr in onlyconfig
* [ ] Graceful shutdown

### Misc

* [ ] Combine dns module and http module

## 4. User Guide

### Dynamic configure DNS records via web manager

Make sure the following items are created on onlyconfig web manager

```text
application: nekoq-bootstrap
environment: PROD
datacenter: default
namespace: nekoq-bootstrap.dns
key: records
```

Content template

```text
[A]
"node1.example.dns"="127.0.0.1"

[TXT]
"node1.example.dns"="Hello World"

[SRV]
"node1.example.dns"='{"priority":10,"weight":20,"port":30,"target":"service.node1.example.dns"}'

[PTR]
"8.8.8.8" = 'demo1.example.com'

```

## 5. Design

### Cluster design

Use simple replication model:

```text
Copy local services to every node that requesting the data
```

In this case, when every node in the cluster listens to other nodes, the cluster will reach a consistent state in which
every node has the full data set of the cluster.

In addition, one node can be easily configured to observer mode when it listens to the cluster but nobody else listens
to itself.

However, the drawbacks of this design is:

1. If network splits, it can cause brain split as no consistency protocol runs to guarantee the majority.
2. Data sync may great impact the network infrastructure even when full data sync happens.

### Dependency Principles

* Keep as minimum dependencies as possible

## 6. Changelog

### Planning

* [ ] Refactor HA module

### v0.0.201

* [X] DNS module: multiple dns record types support
* [X] DNS record dynamic loading via nekoq-component/configure and onlyconfig
* [x] Web manager via nekoq-component/configure and onlyconfig

### v0.0.200

* [X] DNS module: dns/http for A record
* [X] Http module: query/register service
* [X] HA module: data sync

## Appendix A. References

## Appendix B. Testing cases

```text
dig @127.0.0.1 -p 8053 node1.example.dns
dig @127.0.0.1 -p 8053 node1.example.dns TXT
dig @127.0.0.1 -p 8053 node1.example.dns SRV
dig @127.0.0.1 -p 8053 8.8.8.8.in-addr.arpa PTR
```
