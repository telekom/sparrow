<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Design: S3 OIDC Authentication via File-Based Token

## Context

Sparrow needs S3-compatible object storage as an alternative target backend
to GitLab. Instances run on AWS, GCP, Azure, on-prem VMs (T-Cloud/OpenStack),
and Kubernetes. Security policy requires keyless authentication via Workload
Identity Federation (WIF) where available, with static credentials as fallback
for on-prem environments.

The S3 interactor uses [minio-go v7](https://github.com/minio/minio-go).
OIDC auth works by exchanging a short-lived identity token for temporary S3
credentials via AWS STS (or a compatible endpoint). The question is how
sparrow should obtain that identity token.

## Decision

Use `tokenPath` (file-based token read) as the only OIDC token source.
Do not implement `tokenURL` or `tokenHeaders` for direct IMDS HTTP calls.

## Why Not a Generic `tokenURL` Fetcher

IMDS endpoints are fundamentally incompatible across providers:

| Cloud      | Token retrieval                                                  | Response format                                       |
| ---------- | ---------------------------------------------------------------- | ----------------------------------------------------- |
| AWS IMDSv2 | Two-step PUT then GET with session token header                  | JSON with `AccessKeyId` / `SecretAccessKey` / `Token` |
| Azure      | Single GET with `Metadata: true` header, resource query param    | JSON `access_token` field                             |
| GCP        | Single GET with `Metadata-Flavor: Google` header, audience param | Raw JWT string                                        |
| Kubernetes | Projected volume file                                            | Raw JWT file                                          |

A `tokenURL` + `tokenHeaders` abstraction would:

- Hide provider-specific multi-step flows (AWS needs PUT before GET)
- Require per-provider response parsing (JSON field vs. raw body)
- Still not handle AWS IMDSv2 session token management correctly
- Create a leaky abstraction that encourages misconfiguration

## Why File-Based Tokens Work Universally

Every platform has a well-established mechanism to project tokens to disk:

- **Kubernetes (all clouds):** Projected service account token volumes —
  a native K8s feature, no cloud dependency
- **AWS EKS:** IRSA and EKS Pod Identity both write tokens to files
- **Azure AKS:** The Workload Identity webhook projects tokens to the pod
  filesystem automatically
- **GCP GKE:** Workload Identity Federation via node metadata proxy
  (transparent); file projection also works when needed
- **VMs / on-prem:** A sidecar or cron job fetches from IMDS and writes to
  a well-known path — an established ops pattern used by tools such as
  `aws-vault`, `aad-pod-identity`, and custom scripts

This pushes cloud-specific IMDS complexity to the operations layer, where
teams already have domain expertise and tooling for their provider. Sparrow's
code stays provider-agnostic: one `os.ReadFile` call, no HTTP client, no
response parsing, no provider-specific headers.

## Configuration

```yaml
targetManager:
  type: s3
  s3:
    endpoint: s3.amazonaws.com
    bucket: sparrow-targets
    auth:
      provider: oidc
      oidc:
        tokenPath: /var/run/secrets/tokens/sparrow
        roleARN: arn:aws:iam::123456:role/sparrow
        stsEndpoint: ""  # optional; defaults to AWS STS global endpoint
```

Static credentials remain available as a fallback for environments where
token projection is not feasible (e.g. T-Cloud bare-metal).

## Implementation

The OIDC credential provider wraps `credentials.NewSTSWebIdentity` from
minio-go. On each credential refresh, sparrow reads `tokenPath` and
exchanges the token with STS for temporary credentials. Token rotation is
handled entirely by the platform (kubelet, sidecar, cron) — sparrow
re-reads the file on every STS exchange and requires no token lifecycle
logic of its own.

## Consequences

- **VM deployments** require a token-refresh sidecar or cron job.
  This is additional ops work but follows established patterns.
- **Simpler code:** credential provider is ~30 lines — read file, call STS.
- **Security:** token files must be mounted read-only (`0400`).
  Kubernetes projected volumes enforce this automatically; VM operators
  must set permissions explicitly.
- **Extensibility:** a `tokenURL` option can be added in a future PR if
  concrete demand emerges from teams that cannot run a sidecar. The config
  structure reserves space for it under `auth.oidc`.

## See Also

- [minio-go STSWebIdentity](https://pkg.go.dev/github.com/minio/minio-go/v7/pkg/credentials#NewSTSWebIdentity)
- [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
- [Azure Workload Identity](https://azure.github.io/azure-workload-identity/docs/)
- [GCP Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation)
- [Kubernetes projected volumes](https://kubernetes.io/docs/concepts/storage/projected-volumes/)
