# Current Envoy prototypes and issue reproductions

These public repositories explore current Envoy concepts or provide focused reproductions for
upstream issues. For SDKs, tooling, and historical examples, see the broader
[Envoy project archive](/notes/envoy/projects).

Issue and pull-request status was checked on 2026-08-14.

## One Route, One Cluster, Many Providers

[`envoy-one-cluster-many-providers`](https://github.com/dio/envoy-one-cluster-many-providers)
composes Envoy's tip-of-tree dynamic-module SDK into request-aware upstream selection. An HTTP
filter records an opaque promise in filter state, resolves a provider decision after inspecting
the request, and lets a cluster-provided load balancer finish host selection asynchronously.

The application owns logical provider identity while Envoy retains connection pooling, retries,
TLS, health, and metrics. The design also keeps logical hostname identity separate from socket
address so `auto_host_sni` and SAN validation can behave correctly.

Read the full article: [One Route, One Cluster, Many Providers](/notes/envoy/one-route-one-cluster-many-providers).

## Credential injection for Envoy callouts

[`envoy-callout-credential-injection`](https://github.com/dio/envoy-callout-credential-injection)
demonstrates that stock Envoy can authenticate both `ext_proc` gRPC and `ext_authz` HTTP callouts
with a generic credential loaded through SDS. The credential injector runs only in each callout
cluster's upstream filter chain, so the secret is not added to the original request sent to the
backend.

This is a runnable answer to open [Envoy issue #41767](https://github.com/envoyproxy/envoy/issues/41767),
using the dual-filter capability merged in
[Envoy PR #38398](https://github.com/envoyproxy/envoy/pull/38398).

## Session affinity for external processors

[`ext-proc-session-affinity`](https://github.com/dio/ext-proc-session-affinity) shows that Envoy's
existing gRPC client and cluster hash policies can keep the same downstream session key on the
same external processor. The demo runs two processors and makes selection observable for both
header-based and cookie-based ring hashing.

It demonstrates the configuration discussed in closed
[Envoy issue #46159](https://github.com/envoyproxy/envoy/issues/46159) and documented by merged
[Envoy PR #46558](https://github.com/envoyproxy/envoy/pull/46558).

## Preserve the original local reply body

[`envoy-custom-response-local-reply-body`](https://github.com/dio/envoy-custom-response-local-reply-body)
is a configuration-only reproducer for custom response formatting. It isolates whether
`%LOCAL_REPLY_BODY%` can use the body that originally produced an Envoy local reply while keeping
an explicitly configured policy body authoritative.

The behavior is tracked by open [Envoy issue #45346](https://github.com/envoyproxy/envoy/issues/45346).
The candidate patch and its focused test cases are in open
[Envoy PR #46555](https://github.com/envoyproxy/envoy/pull/46555).

## Logical hostnames for runtime-selected upstreams

[`auto-sni-choose-host`](https://github.com/dio/auto-sni-choose-host) is a minimal dynamic-module
cluster that selects HTTPS hosts from a request header. It demonstrates why a runtime host needs a
logical hostname in addition to its concrete socket address for `auto_host_sni` and
`auto_sni_san_validation`.

The work is tracked in open [Envoy issue #45962](https://github.com/envoyproxy/envoy/issues/45962).
Logical-hostname support landed in [Envoy PR #46388](https://github.com/envoyproxy/envoy/pull/46388),
and SNI-scoped session caching landed separately in
[Envoy PR #45982](https://github.com/envoyproxy/envoy/pull/45982).

## Cross-SNI TLS session-cache reproducer

[`envoy-sni-session-cache-repro`](https://github.com/dio/envoy-sni-session-cache-repro) sends two
requests to different hostnames that resolve to the same address and present different
certificates. The affected build incorrectly offers the first hostname's cached TLS session to the
second; the candidate build scopes cached sessions by SNI and completes a fresh handshake.

The repository reproduces open [Envoy issue #46243](https://github.com/envoyproxy/envoy/issues/46243)
and compares the affected release with the fix merged in
[Envoy PR #45982](https://github.com/envoyproxy/envoy/pull/45982).

## HTTP transcoder EOF over HTTP/2

[`envoy-16629`](https://github.com/dio/envoy-16629) is a compact Docker Compose reproducer for a
transcoder behavior difference. The same request produces the expected gRPC internal error over
HTTP/1.1 but becomes an unexpected EOF response over prior-knowledge HTTP/2.

The original report, [Envoy issue #16629](https://github.com/envoyproxy/envoy/issues/16629), is
closed; the repository remains useful as a historical regression scenario.
