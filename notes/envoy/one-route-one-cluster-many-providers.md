# One Route, One Cluster, Many Providers

## Request-aware upstream selection with Envoy dynamic modules

> Source and prototype: [dio/envoy-one-cluster-many-providers](https://github.com/dio/envoy-one-cluster-many-providers).

### Abstract

Recent Envoy dynamic-module work enables an interesting deployment shape: one route and one dynamic
cluster can serve many logical providers. An HTTP extension can parse or look up a request decision,
store a `selected_backend` or `selected_endpoint` value in request state, and let the cluster consume
that value while choosing the upstream host.

The route does not need to be selected again. The decision does not need to travel through a
synthetic request header. Envoy still owns the selected host, connection pool, retries, health, TLS,
and metrics. The extension only supplies the application-level identity that Envoy could not know
from route configuration alone.

There is one complication. Envoy may ask the cluster to choose a host before the extension has
finished parsing the body or consulting an external directory. Filter state solves the data handoff,
but not this timing mismatch. Asynchronous host selection solves the timing mismatch. The rest of
this article develops the small per-stream coordination point needed to compose the two.

## The clue was in the func-e release notes

I was reading the
[func-e v1.6.0 release notes](https://github.com/tetratelabs/func-e/releases/tag/v1.6.0) for the new
Envoy development-build support when one example jumped out: a dynamic module can write
`selected_backend` or `selected_endpoint` into request state, and the cluster can use that value
while choosing a host.

That one observation opens a much larger design space. A dynamic module can derive either a logical
backend or a specific endpoint from a request and make the selection available exactly where Envoy
needs it: host-selection time.

`selected_backend` and `selected_endpoint` are alternative application-defined keys, not special
Envoy fields and not a pair that must be written together:

- `selected_backend` can name a provider or logical service that the cluster resolves through its
  catalog.
- `selected_endpoint` can name a concrete endpoint when an earlier extension has already completed
  placement.

This turns request interpretation into an input to load balancing without turning it into another
route match.

## One stable route and cluster

Consider a gateway that accepts a common API for several model providers:

```text
POST /v1/chat/completions
```

The JSON body identifies a model. Tenant policy and catalog data map that model to a provider. Each
provider may have a different address, logical hostname, credentials, health state, and connection
pool.

A conventional configuration can encode those choices as many routes and clusters. That works, but
it moves an application catalog into xDS and couples route configuration to every provider change.
The dynamic-module path allows a different split:

1. One Envoy route selects one cluster.
2. The cluster owns a set of Envoy runtime hosts.
3. An HTTP extension derives `selected_backend` or `selected_endpoint` from the request.
4. The cluster-provided load balancer maps that identity to an exact Envoy host handle.
5. Envoy continues through its normal upstream connection path.

```text
request                   application decision                  Envoy upstream
   |                               |                                   |
   +--> one route --> one cluster  |                                   |
                                   +--> exact HostHandle --------------+
                                                                       |
                                                           pool, retry, connect,
                                                           TLS, health, metrics
```

The route and cluster stay stable. Provider identity lives in the extension-owned catalog where the
application can reason about it directly.

## How request state reaches host selection

This works because Envoy now connects two extension phases that were previously difficult to
compose.

On the HTTP side, a dynamic module can write bytes into per-request filter state. Envoy stores those
bytes as a `Router::StringAccessor` with filter-chain lifetime. The value remains attached to the
stream rather than becoming part of the HTTP protocol.

On the upstream side, the router calls `cluster->chooseHost()` with a `LoadBalancerContext` for that
same stream. The dynamic-module cluster's
[per-request context API](https://github.com/envoyproxy/envoy/pull/43858) exposes the context to its
worker-local load balancer. The
[cluster filter-state read API](https://github.com/envoyproxy/envoy/pull/45040) then reaches through
`requestStreamInfo()` to the same filter state written by the HTTP extension.

When the decision is already available, the flow is direct:

```text
HTTP extension             request filter state             cluster-provided LB
parse or look up  -------> selected_backend -----------> catalog lookup -> HostHandle
```

No synthetic `x-selected-backend` header is required. There is no risk that an internal routing
instruction is forwarded upstream, stripped by an intermediary, or confused with client input. No
route recreation is required either. Route selection has already done its job by choosing the
dynamic cluster.

## Why not Dynamic Forward Proxy?

Sometimes [Dynamic Forward Proxy](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/http/http_proxy)
is the better answer. If the selection is fundamentally a DNS hostname and port, DFP already owns
the right machinery: asynchronous DNS resolution, a shared DNS cache, host TTLs, circuit breakers,
and automatic SNI and SAN verification.

DFP can also take its target from filter state. With
[`allow_dynamic_host_from_filter_state`](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/dynamic_forward_proxy_filter#dynamic-host-resolution-via-filter-state)
enabled, a preceding filter can write `envoy.upstream.dynamic_host` and
`envoy.upstream.dynamic_port` before the DFP filter processes the request. If application policy can
produce a hostname at that point, using DFP is simpler than maintaining a custom host catalog and
load balancer.

This prototype addresses a different boundary. Its selection may be a provider ID rather than a DNS
name, or an exact endpoint already registered with Envoy. An external catalog may own placement,
weights, health, and logical hostname independently from DNS. Most importantly, the decision may
arrive after DFP and the router have processed request headers. A filter could stop iteration and
buffer until the decision is ready, then hand a hostname to DFP, but that deliberately changes the
streaming and latency behavior.

The cluster-provided approach instead keeps host selection pending and later completes it with an
exact Envoy host handle. It gives the extension more flexibility, but also makes the extension
responsible for catalog consistency, host lifecycle, cancellation, and safe removal.

A useful rule of thumb is:

- Use DFP when the answer is `hostname:port` and Envoy should own discovery through DNS.
- Use a cluster-provided load balancer when the answer is application identity or exact placement
  and the application already owns discovery.
- Combine them when policy can resolve a hostname before DFP runs: let the extension decide the
  hostname, then let DFP own DNS and transport behavior.

## The remaining problem is time

The direct flow assumes that the decision exists before `chooseHost()` runs. That is true for a
header, cookie, or already-populated session key. It is not necessarily true for a request body or
an asynchronous lookup.

At headers time, Envoy knows the route and can begin upstream selection. At body time, the HTTP
extension may finally know the provider. This creates the actual problem:

> Envoy must choose a host before the application knows which host to choose.

Writing filter state later is not sufficient by itself. The cluster has already been asked for a
host, and Envoy does not poll an application key waiting for it to appear. Buffering the complete
body before continuing can avoid the race, but changes latency, memory use, and streaming behavior.

What is needed is a coordination point between request understanding and host selection.

## Illustrative use case: resume a stateful inference session

Imagine an inference service where a processor keeps expensive session state in memory, such as a
conversation workspace, an agent execution, or a model cache. A later request should return to that
processor while the state is live. The example API is illustrative rather than tied to a particular
provider:

```http
POST /v1/inference
content-type: application/json

{
  "model": "reasoning-large",
  "session": "session-7f2a",
  "input": "continue the analysis"
}
```

The route is always `/v1/inference`, and every processor belongs to one dynamic cluster. The
request body supplies the session identity, but an external session directory owns the current
placement:

```text
session-7f2a -> processor-17 -> HostHandle(17)
```

The request can flow as follows:

1. At headers time, the HTTP extension creates a pending per-stream decision and lets Envoy
   continue.
2. Envoy reaches the cluster, which finds no completed decision and returns async pending.
3. The HTTP extension parses `session-7f2a` from the body and asks the session directory where its
   state lives.
4. The directory responds with `processor-17`. The extension resolves the selection to that
   endpoint identity.
5. The cluster maps `processor-17` to `HostHandle(17)` and completes host selection.
6. Envoy reuses or creates the pool for that host and sends the request directly to the processor
   holding the session.

If the session has no owner, policy may instead resolve a `selected_backend` so the cluster can
choose among processors able to create it. If the lookup times out, the promise resolves to a
no-host result rather than leaving the stream suspended indefinitely.

This avoids an affinity header that every intermediary must understand. It also avoids routing
through another load balancer just to repeat the placement lookup. The session directory remains
outside Envoy, while Envoy remains responsible for the concrete upstream connection.

## Promise now, route when ready

Envoy's [asynchronous host-selection API](https://github.com/envoyproxy/envoy/pull/43859) lets a
cluster return a pending selection rather than guessing or failing immediately. The router retains a
cancellation handle and postpones connection-pool creation until the module completes with a host or
a no-host result.

That is where `StreamPromise[T]` enters the design. It is not a new routing abstraction. It is a
typed, resolve-once coordination primitive scoped to one stream.

At headers time, the HTTP extension creates the promise and writes only an opaque correlation token
to filter state:

```go
type Decision struct {
	Provider string
	Model    string
	Err      string
}

token, decision, err := decisions.Create()
if err != nil {
	return err
}
handle.SetFilterState("demo.provider-promise", []byte(token))
```

When the cluster is asked to choose a host, it reads the token and subscribes to the corresponding
module-owned promise:

```go
pending, ok := selector.Select(token, completion)
if !ok {
	return missingDecision()
}
return asyncPending(pending)
```

Later, body parsing or an external lookup resolves the typed decision:

```go
decision.Resolve(Decision{Provider: "provider-a", Model: "model-a"})
```

The selector maps `provider-a` to an exact host handle and completes Envoy's pending selection.

```text
HTTP extension         Envoy router          cluster-provided LB        upstream
      |                      |                         |                     |
      | create StreamPromise |                         |                     |
      | publish opaque token |                         |                     |
      |   continue headers   |                         |                     |
      |--------------------->|                         |                     |
      |                      |       ChooseHost        |                     |
      |                      |------------------------>|                     |
      |                      |      async pending      |                     |
      |                      |<------------------------|                     |
      | parse body or lookup |                         |                     |
      |   resolve Decision   |                         |                     |
      |----------------------------------------------->|                     |
      |                      |      complete host      |                     |
      |                      |<------------------------|                     |
      |                      | create or reuse pool    |                     |
      |                      |---------------------------------------------->|
```

Filter state carries a safe byte-oriented correlation value. The typed decision and host handle stay
inside module memory. No Go pointer crosses Envoy's ABI.

The [async request-context API](https://github.com/envoyproxy/envoy/pull/45495) also makes the
request context available when an asynchronous selection resumes. Completion may originate on
another thread, so Envoy snapshots cancellation state, posts the result to the request worker,
recovers a live shared host from the opaque handle, and invokes `onAsyncHostSelection`. If the stream
has already ended, cancellation prevents late completion.

Filter state therefore solves the data problem. Async host selection solves the timing problem.
`StreamPromise[T]` is the small layer that joins them when the data arrives late.

## Host identity must remain distinct from address

The extension should resolve application identity to the exact host handle Envoy returned when the
runtime host was registered. It should not rediscover a host from its socket address:

```text
provider-a -> HostHandle(1) -> 203.0.113.10:443, a.example
provider-b -> HostHandle(2) -> 203.0.113.10:443, b.example
```

[Envoy PR #46388](https://github.com/envoyproxy/envoy/pull/46388) added an opt-in cluster callback
that assigns a logical hostname separately from the connection address. That hostname is then
available to existing upstream features such as `auto_host_sni` and `auto_sni_san_validation`.

That change supplies the identity, but a shared TLS context also has to preserve it during session
resumption. I had previously fixed that side in
[Envoy PR #45982](https://github.com/envoyproxy/envoy/pull/45982). Before the fix, upstream TLS
sessions were cached at the `ClientContextImpl` level. A session learned while connecting with one
effective SNI could therefore be offered on a later connection using another effective SNI.

PR #45982 scopes the upstream session cache by the effective SNI, using the same precedence as the
ClientHello: an explicit transport-socket override, the host hostname selected by `auto_host_sni`,
then the static upstream TLS SNI. With one shared `UpstreamTlsContext`, a connection to `a.example`
can reuse a session learned for `a.example`, but not one learned for `b.example`.

The two changes complete the TLS story for this design:

```text
#46388: selected HostHandle -> logical host hostname -> auto_host_sni
#45982: effective SNI      -> isolated TLS session-cache key
```

Together, they let one dynamic cluster use one shared TLS context across multiple logical upstream
hosts while preserving the selected host's SNI identity for both certificate validation and session
resumption.

Envoy currently still deduplicates dynamic-module runtime hosts by IP and port. Two logical hosts
sharing one socket therefore remain a focused follow-up, not a capability claimed by this prototype.
The important invariant is already visible: provider identity maps to an Envoy-owned handle, not
back to an ambiguous address string.

## The session-affinity connection

Ordinary hash policy remains the simpler answer when an affinity key is available early and the
selected upstream is the final placement authority.

The limitation appears when that upstream is another load-balancing proxy. End-to-end affinity then
depends on every intermediary preserving and applying the same key. This concern appeared directly
in the discussion following
[#46558](https://github.com/envoyproxy/envoy/pull/46558#issuecomment-5272902090).

Late host selection offers another option when the gateway can resolve final placement itself:

- decode a composite session and find its owning processor;
- resolve an MCP tool to its server and provider;
- consult an asynchronous placement directory;
- derive a provider from a request body; or
- combine tenant policy, model identity, and health into one typed decision.

The extension selects the concrete final host instead of relying on another proxy to repeat the
placement decision. The promise does not make intermediaries affinity-aware. It can make an
intermediary unnecessary for this step.

## A thin Go layer over the upstream SDK

The prototype is intentionally a composition layer, not a replacement SDK and not a gateway
framework. The HTTP side imports Envoy's tip-of-tree Go dynamic-module SDK directly. The additional
pieces are small:

- `StreamPromise[T]` provides a typed first-resolution-wins value.
- `Registry[T]` correlates an opaque token with module-owned state.
- `Selector[T]` owns exact-once completion and cancellation.
- `provider.Manager` publishes one complete initial provider-to-host-handle snapshot.

The upstream Go SDK does not yet expose the cluster extension API. Until it does, a narrow cgo seam
is needed for the cluster callbacks: add hostname-aware hosts, update health, read filter state,
schedule cluster work, and complete asynchronous host selection. When Envoy gains a native Go
cluster API, that seam can disappear without changing the promise, catalog, or application filter.

The difficult part of this wrapper is lifecycle rather than syntax:

- the first terminal resolution wins;
- cancellation prevents late completion;
- stream teardown removes the registry entry;
- initial catalog publication exposes only valid handles;
- live removal waits for a future drain or grace protocol; and
- cluster mutations run on Envoy's required thread.

## Trying the merged capability before the next release

This brings the story back to the func-e release notes that exposed the use case. Envoy releases are
worth the wait, but they land roughly once a quarter. The development build covers the interval
between a feature being merged and appearing in a tagged release.

func-e 1.6.0 added `dev` and `dev-latest` resolution, so current Envoy mainline can be tested without
building Envoy locally or relying on a Linux-only Docker workflow:

```sh
ENVOY_VERSION=dev func-e run --version
```

The archived macOS ARM64 development build dated 2026-08-12 reports Envoy `1.40.0-dev` at commit
`b7c9126475069d4aa184dcfc3615ec69f9b0d81d`. That commit includes the merged hostname-aware host ABI
from #46388. The current artifact and SHA-256 are published in the
[`dev` archive index](https://archive.tetratelabs.io/envoy/envoy-versions.json), with binaries on the
[`dev` release](https://github.com/tetratelabs/archive-envoy/releases/tag/dev).

Use a development build to exercise an unreleased ABI, validate upgrade impact early, or provide
feedback before the next release is cut. Use tagged Envoy versions for normal production runs.

This repository currently implements the HTTP and generic coordination layers against the upstream Go
SDK. The narrow Go cluster ABI adapter is the next slice, so this is an architecture prototype rather
than an end-to-end runnable module today.

## Closing

The new possibility is larger than body-based routing. An extension can turn any request-derived
identity into an input to host selection while Envoy continues to own proxy mechanics.

When the identity is ready, request state is enough. When the identity arrives late, asynchronous
host selection gives Envoy a safe way to wait. `StreamPromise[T]` makes that wait explicit, typed,
and cancellation-aware.

One route. One dynamic cluster. Many providers.

Promise now. Route when ready.
