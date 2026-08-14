# RockyBars Notes

Source for [rockybars.com](https://rockybars.com/) and technical things worth
keeping.

The site is a small Go application that preserves the existing RockyBars
landing page and publishes Markdown from [`notes/`](notes/) at `/notes`.

## Run locally

```console
go run .
```

The server listens on `PORT`, defaulting to `8080`.

## Deployment

Pull requests run tests, static analysis, and a container build. Every push to
`main` that passes those checks deploys to the `rockybars-mainsite` Fly app and
verifies the public `/notes` endpoint. Deployment uses the app-scoped
`FLY_API_TOKEN` GitHub Actions secret; rotate that deploy token annually.

## Envoy

- [ADS `DiscoveryRequest.version_info` across stream reconnects](notes/envoy/ads-discovery-request-version-info.md)
- [One Route, One Cluster, Many Providers](notes/envoy/one-route-one-cluster-many-providers.md)
  ([source prototype](https://github.com/dio/envoy-one-cluster-many-providers))
- [Current Envoy prototypes and issue reproductions](notes/envoy-prototypes-and-issues.md)
- [Envoy project archive](notes/envoy/projects.md)
