# Envoy projects and issue reproductions

This is a curated inventory of public, original repositories under
[`github.com/dio`](https://github.com/dio) that explore Envoy concepts, reproduce issues, or make
Envoy development easier. It was reviewed on 2026-08-14. Forks, binary mirrors, private
repositories, and unrelated products are intentionally omitted.

Older experiments may use historical Envoy APIs or container images. Treat them as references,
not as promises of compatibility with current Envoy releases.

## Current concept prototypes and issue reproductions

- [`envoy-one-cluster-many-providers`](https://github.com/dio/envoy-one-cluster-many-providers) —
  request-aware upstream selection with one stable route and dynamic cluster. Read the
  [full article](/notes/envoy/one-route-one-cluster-many-providers).
- [`envoy-callout-credential-injection`](https://github.com/dio/envoy-callout-credential-injection) —
  `ext_proc` and `ext_authz` callout credential-injection demo for
  [Envoy issue #41767](https://github.com/envoyproxy/envoy/issues/41767), using the capability from
  [Envoy PR #38398](https://github.com/envoyproxy/envoy/pull/38398).
- [`ext-proc-session-affinity`](https://github.com/dio/ext-proc-session-affinity) — external
  processing session-affinity demo related to
  [Envoy issue #46159](https://github.com/envoyproxy/envoy/issues/46159) and the merged
  [session-affinity documentation PR #46558](https://github.com/envoyproxy/envoy/pull/46558).
- [`envoy-custom-response-local-reply-body`](https://github.com/dio/envoy-custom-response-local-reply-body) —
  minimal reproducer for [Envoy issue #45346](https://github.com/envoyproxy/envoy/issues/45346),
  with the proposed fix in [Envoy PR #46555](https://github.com/envoyproxy/envoy/pull/46555).
- [`auto-sni-choose-host`](https://github.com/dio/auto-sni-choose-host) — proof of concept around
  logical hostnames, `auto_host_sni`, and runtime-selected hosts, tracking
  [Envoy issue #45962](https://github.com/envoyproxy/envoy/issues/45962),
  [logical-hostname PR #46388](https://github.com/envoyproxy/envoy/pull/46388), and
  [SNI session-cache PR #45982](https://github.com/envoyproxy/envoy/pull/45982).
- [`envoy-sni-session-cache-repro`](https://github.com/dio/envoy-sni-session-cache-repro) — focused
  reproducer for TLS session reuse across different SNI names,
  [Envoy issue #46243](https://github.com/envoyproxy/envoy/issues/46243), and its fix in
  [Envoy PR #45982](https://github.com/envoyproxy/envoy/pull/45982).
- [`envoy-16629`](https://github.com/dio/envoy-16629) — local reproducer for
  [Envoy issue #16629](https://github.com/envoyproxy/envoy/issues/16629).

## Dynamic modules and extension SDKs

- [`luwes`](https://github.com/dio/luwes) — zero-allocation Go SDK and drop-in replacement for
  Envoy's upstream dynamic-modules SDK.
- [`transit`](https://github.com/dio/transit) — an ergonomic Go layer for Envoy dynamic modules.
- [`jisr`](https://github.com/dio/jisr) — Go middleware-style API for Envoy dynamic modules.
- [`zig-envoy-sdk`](https://github.com/dio/zig-envoy-sdk) — Zig SDK for Envoy dynamic-module
  filters.
- [`jisr-hello`](https://github.com/dio/jisr-hello) — archived minimal dynamic-module filter
  example.

## Build, test, and operations helpers

- [`envoy-builder`](https://github.com/dio/envoy-builder) — on-demand Envoy builds using GitHub
  Actions.
- [`envoy-mini-builder`](https://github.com/dio/envoy-mini-builder) — remote Mac mini builds and
  releases for Envoy.
- [`setup-envoy`](https://github.com/dio/setup-envoy) — GitHub Action for installing Envoy.
- [`envoy-test-tools`](https://github.com/dio/envoy-test-tools) — utilities for Envoy-focused
  tests.
- [`gh-istio-envoy-version`](https://github.com/dio/gh-istio-envoy-version) — GitHub CLI extension
  for inspecting Istio and Envoy version relationships.
- [`envoy-docker-compose`](https://github.com/dio/envoy-docker-compose) — Docker Compose environment
  for working with Envoy.
- [`gateway-pairs`](https://github.com/dio/gateway-pairs) — Envoy Gateway namespace-pair
  experiments.

## Earlier experiments and examples

- [`per-route-envoy-lua`](https://github.com/dio/per-route-envoy-lua) — experimental per-route Lua
  configuration.
- [`envoy-lua-opa`](https://github.com/dio/envoy-lua-opa) — Open Policy Agent authorization from
  an Envoy Lua filter using `httpCall`.
- [`envoy-lua-test`](https://github.com/dio/envoy-lua-test) — LuaSocket experiment inside Envoy's
  Lua filter; the blocking approach is not recommended for production.
- [`authz`](https://github.com/dio/authz) — `ext_authz` experiment combined with dynamic forward
  proxying.
- [`envoy-mysql-ext_authz-test`](https://github.com/dio/envoy-mysql-ext_authz-test) — MySQL proxy
  metadata and external-authorization experiment.
- [`ruby-als-server`](https://github.com/dio/ruby-als-server) — Ruby access-log service example.
- [`envoy-filter-diy`](https://github.com/dio/envoy-filter-diy),
  [`envoy-diy-filter-1`](https://github.com/dio/envoy-diy-filter-1), and
  [`envoy-filter-example-1`](https://github.com/dio/envoy-filter-example-1) — C++ HTTP filter
  examples from different iterations.
- [`simple-grpc`](https://github.com/dio/simple-grpc) — gRPC topology with front and internal
  Envoys, TLS, and health checking.
- [`envoy-hello-validation-context`](https://github.com/dio/envoy-hello-validation-context) —
  client-certificate validation and routing example.
- [`envoy-authorization-header-check-js`](https://github.com/dio/envoy-authorization-header-check-js) —
  authorization-header checking experiment in JavaScript.
