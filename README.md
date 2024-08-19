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
* [X] DNS service discovery: TXT record
* [X] DNS upstream name server support
* [X] DNS over http - rfc8484
* [ ] DNS over https - rfc8484
* [ ] DNS service discovery - AAAA/MX/SRV/CNAME
* [ ] DNS service discovery - SOA/PTR
* [ ] DNS Sec
* [ ] DNS TCP
* [ ] Recursive DNS
* [ ] Authority DNS Server
* [ ] DNS caching and TTL
* [ ] DNS recursion support
* [ ] DNS Authoritative

### Http module

* [X] Register several types of service
  * Support register same node to several NekoQ-Bootstrap. Note: DO NOT use different data in this case. Otherwise only the latest registration will be effect under current HA strategy within the cluster.
* [X] Peer auth

### High available cluster module

* [X] Peer data sync
* [X] Peer auth
* [ ] Peer data sync: dns data

### Simple KV store module

* [ ] KV store

### Management

* [ ] web manager
* [ ] Graceful shutdown

### Misc

* [ ] Combine dns module and http module

## 4. Design

### Cluster design

Use simple replication model:

```text
Copy local services to every node that requesting the data
```

In this case, when every node in the cluster listens to other nodes, the cluster will reach a consistent state in which every node has the full data set of the cluster.

In addition, one node can be easily configured to observer mode when it listens to the cluster but nobody else listens to itself.

However, the drawbacks of this design is:

1. If network splits, it can cause brain split as no consistency protocol runs to guarantee the majority.
2. Data sync may great impact the network infrastructure even when full data sync happens.

### Dependency Principles

* Keep as minimum dependencies as possible

## 5. Changelog

### Planning
* [ ] Refactor HA module
* [ ] Web manager

### v0.0.200
* [X] DNS module: dns/http for A record
* [X] Http module: query/register service
* [X] HA module: data sync

