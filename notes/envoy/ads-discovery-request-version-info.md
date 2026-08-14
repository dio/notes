# ADS `DiscoveryRequest.version_info` across stream reconnects

In Envoy's state-of-the-world Aggregated Discovery Service (ADS), the
`version_info` field represents the most recent resource version that Envoy
successfully accepted. It belongs to the resource state, not to an individual
gRPC stream.

Therefore, "empty on the first request" in the
[`DiscoveryRequest` documentation][discovery-request] means the first request
for a resource type before Envoy has accepted any response. It does **not**
mean the first request on every newly created gRPC stream.

## Reconnection example

```text
ADS stream A:
  CDS request:  version_info="",   response_nonce=""
  CDS response: version_info="42", nonce="a"
  CDS ACK:      version_info="42", response_nonce="a"

stream A disconnects

ADS stream B:
  CDS request:  version_info="42", response_nonce=""
```

The fields have different lifetimes:

- `version_info` describes Envoy's accepted resource state and survives a
  stream reconnection.
- `response_nonce` identifies a response on one particular stream and does
  not survive a stream reconnection.
- In ADS, both values are tracked independently for each `type_url`.

The [xDS protocol documentation][xds-version] states that a resource version
is not a property of an individual stream. The initial request on a replacement
stream should contain the most recent version seen on the previous stream.

Envoy's implementation follows that rule. When a SotW stream becomes fresh,
Envoy clears the last good nonce but retains the last good version. It then
places that retained version in the next `DiscoveryRequest`.

## Rolling control-plane upgrades

After a control-plane replica is replaced, the next replica may receive an
initial request with an existing `version_info` and an empty `response_nonce`.
That is a normal reconnect, not a NACK.

Replicas serving one logical xDS `ConfigSource` should:

- use a shared or deterministic version namespace for each resource type;
- reconstruct subscription state from the new stream instead of relying on
  state held by the previous replica;
- send the latest SotW snapshot when its version differs from the version
  reported by Envoy; and
- treat nonces as stream-local values.

If Envoy reports version `42` and the current snapshot is still version `42`,
the server may avoid resending it when doing so is safe for the subscription.
If the current snapshot is version `43`, the server sends the full version
`43` snapshot and Envoy ACKs or NACKs it normally.

Different control-plane releases can generate different version strings for
identical configuration without violating correctness: the new replica sends
its version and Envoy processes it. Stable, deterministic versions are still
preferable because they avoid unnecessary configuration reprocessing during
a rollout.

## Sources

- [Envoy `DiscoveryRequest` API documentation][discovery-request]
- [Envoy xDS ACK/NACK and resource-version documentation][xds-version]
- [Envoy SotW subscription implementation][sotw-implementation]

[discovery-request]: https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/discovery/v3/discovery.proto#service-discovery-v3-discoveryrequest
[xds-version]: https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#ack-nack-and-resource-type-instance-version
[sotw-implementation]: https://github.com/envoyproxy/envoy/blob/main/source/extensions/config_subscription/grpc/xds_mux/sotw_subscription_state.cc
