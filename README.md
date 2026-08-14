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

## Envoy

- [ADS `DiscoveryRequest.version_info` across stream reconnects](notes/envoy/ads-discovery-request-version-info.md)
