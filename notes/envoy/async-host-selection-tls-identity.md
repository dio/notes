# When does async host selection need a TLS options refresh?

*Research notes, 2026-09-08.*

After [#47260](https://github.com/envoyproxy/envoy/pull/47260) merged, I revisited
[the tracking issue for dynamic-module hostnames and TLS](https://github.com/envoyproxy/envoy/issues/45962).
The combined test now covers async host selection, logical-hostname SNI, SAN validation, and
TLS session reuse. Two questions from the original prototype still deserve separate treatment:
worker membership during selection, and TLS overrides written while selection is pending.

The useful question is whether an application actually needs to learn its TLS identity that late.
That is plausible, but choosing a host asynchronously does not by itself require a TLS options
refresh. The distinction is where the identity lives and when Envoy captures it.

## A plausible application requirement

Consider a gateway with one route and one dynamic cluster. A request identifies a tenant and model.
An asynchronous placement service returns both the destination and the TLS name expected there:

```text
tenant + model -> placement lookup
              -> address: 192.0.2.10:443
              -> TLS name: tenant-a.inference.example
```

The address might serve several tenant names. Alternatively, placement might select a regional
provider with a different TLS name. The gateway cannot finish deciding how to connect until the
lookup returns. This is an illustrative deployment, not a production incident reproduced here.

There are three ways to express that result:

- **The name belongs to a logical host.** Register that hostname on the Envoy host and return its
  exact handle. With `auto_host_sni` and the corresponding SAN validation configured, host selection
  supplies the TLS identity. The name can become known during discovery without becoming a late
  request-specific override. Multiple logical hosts at one address also require a host catalog
  that supports that representation; #47260 does not establish that capability.
- **The name is a request option that can be resolved before the router.** An HTTP filter can pause
  the request, finish the lookup, populate the appropriate typed filter-state objects, and continue.
  Envoy's [Dynamic Forward Proxy][dfp] and [on-demand discovery][on-demand] already illustrate
  waiting in an HTTP filter before forwarding. This need not mean buffering the entire body:
  retain only what the decision requires, within the filter's buffering and timeout rules.
- **The name is a request option learned inside an already-pending host selection.** If the
  application deliberately puts the lookup there, and returns an existing host plus a distinct
  request-specific SNI or SAN override, the router's earlier snapshot becomes relevant.

That third case is a reasonable extension design to investigate. It is a narrower requirement than
"routing depends on an async lookup." Before changing Envoy, I want a concrete reason why the
first two representations are insufficient for the application.

## What the router captures

In the [upstream revision inspected][router-snapshot], the router constructs
`transport_socket_options_` from request filter state before calling `chooseHost()`.
The [conversion][from-filter-state] copies the SNI, SAN, and ALPN override values into an options
object. Writing new filter state later does not update those copies.

The [async completion path][async-completion] creates the connection pool without rebuilding the
options. A proposed late-override flow therefore looks like this:

```text
router captures TLS options       no explicit override
router calls chooseHost()         selection becomes pending
lookup completes on the worker    add SNI/SAN overrides to request state
selection completes               return the selected host
router obtains a connection pool  still supplies the earlier options
```

This is a source-derived stale-snapshot scenario. We have not yet run a regression test proving
that late-write path. A module also needs an API that creates the typed objects consumed by
`fromFilterState`; writing a generic string under the same key is not sufficient evidence.

If this behavior is to be supported, refreshed options must reach the router before connection-pool
lookup. SNI, SAN, and ALPN overrides participate in [pool-key hashing][pool-key]. Updating only
the eventual TLS handshake would be too late to make the pool choice reflect the new options.
The contract also needs to say what happens on retries and which overrides may change.

## What the merged test proves

The hosts in #47260 already carry their logical hostnames. The asynchronous result supplies the
selected host, and TLS derives SNI from that host. There is no late request-specific TLS override.

The tests select A/B/A/B and B/A/B/A under one upstream TLS context. Both servers share a
certificate and ticket key, so server-side ticket separation cannot hide cross-name reuse.
After each response, the test closes the upstream connection and waits for its destruction.
Every request must therefore make a new connection and perform a TLS handshake.

For either ordering, the cumulative counters must be:

```text
request                  first A   first B   second A   second B
TLS handshakes                  1         2          3          4
TLS sessions reused             0         0          1          2
```

The reverse test swaps A and B. The first visit to each name establishes its own session;
returning to each name resumes that name's session. The tests also check which server received
the request and which SNI it observed, while downstream authority remains constant. A separate
case rejects a hostname outside the certificate SANs. The fixture uses TLS 1.2 for deterministic
ticket delivery.

That validates the combination of [logical-hostname support][hostname-pr] and
[SNI-scoped session caching][cache-pr] through async selection. It does not test late TLS overrides,
host replacement during selection, or TLS 1.3 ticket timing.

## What can stay inside the module?

The module can own application identity, placement, and its catalog of exact Envoy host handles.
An HTTP module can also own the wait before the router captures request options. Those are useful
ways to avoid needing a router change, provided they meet the application's requirements.

An internal module variable cannot refresh an options object already owned by the router. If the
required contract is specifically "change request TLS overrides while router host selection is
pending," Envoy needs a way to incorporate the final values before pool lookup.

Worker-local host resolution is a separate concern. Workers have their own priority-set containers,
but normal membership publication [shares the host objects][shared-hosts]. A host created on the
main thread is not inherently a different TLS identity from the host visible to a worker.
The unresolved question is which hosts remain eligible when membership changes during a pending
selection.

A module can track worker membership and define its pending-selection policy. It should preserve
the exact selected identity. The original prototype's address-only fallback is not an adequate
substitute: an old host and its replacement can share an address while carrying different names.
Removal, publication ordering, cancellation, and replacement need explicit behavior and tests.

## The next useful proof

I would start with a router unit test: capture options with no overrides, return pending from
host selection, add typed SNI/SAN state on the request worker, then complete selection and inspect
the options supplied to pool creation. Establish the current failure before adding a refresh.

An integration test should then give the selected host one hostname and the late explicit override
another. That prevents `auto_host_sni` from accidentally satisfying the assertion. Check observed
SNI, certificate validation, pool separation, and session reuse across alternating overrides.
Keep membership-change tests separate, and preserve early overrides, cancellation, and retry
behavior.

The earlier [one-route, many-providers article][providers] develops the application-level handoff.
This follow-up identifies the extra requirement that would justify a router change. The merged
host-driven TLS path already has combined coverage; the late-override path remains a research
question until the narrower contract and failing test are in place.

[dfp]: https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/http/http_proxy.html
[on-demand]: https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/on_demand_updates_filter
[router-snapshot]: https://github.com/envoyproxy/envoy/blob/1b6b34b9d6f1b5980acc0e3d32892d920ee98536/source/common/router/router.cc#L712
[from-filter-state]: https://github.com/envoyproxy/envoy/blob/1b6b34b9d6f1b5980acc0e3d32892d920ee98536/source/common/network/transport_socket_options_impl.cc#L56
[async-completion]: https://github.com/envoyproxy/envoy/blob/1b6b34b9d6f1b5980acc0e3d32892d920ee98536/source/common/router/router.cc#L784
[pool-key]: https://github.com/envoyproxy/envoy/blob/1b6b34b9d6f1b5980acc0e3d32892d920ee98536/source/common/network/transport_socket_options_impl.cc#L22
[hostname-pr]: https://github.com/envoyproxy/envoy/pull/46388
[cache-pr]: https://github.com/envoyproxy/envoy/pull/45982
[shared-hosts]: https://github.com/envoyproxy/envoy/blob/1b6b34b9d6f1b5980acc0e3d32892d920ee98536/source/common/upstream/upstream_impl.cc#L979
[providers]: /notes/envoy/one-route-one-cluster-many-providers
