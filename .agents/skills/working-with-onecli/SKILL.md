---
name: working-with-onecli
description: Guide to the OneCLI package including the Client, CredentialProvider, SecretMapper, and SecretProvisioner interfaces, and how they integrate with the Podman runtime
argument-hint: ""
---

# Working with OneCLI

The `pkg/onecli` package provides a typed HTTP client for the OneCLI API, plus three higher-level abstractions that the Podman runtime uses to provision secrets and configure networking for workspace containers.

## Overview

OneCLI is an HTTP proxy service that runs alongside the workspace container. It:

- Intercepts outbound HTTP requests and injects secret values as headers
- Enforces network rules (allow/block/rate-limit per host pattern)
- Exposes `/api/container-config` which the Podman runtime reads to inject proxy environment variables and a CA certificate into the workspace container

## Key Interfaces

All four public types are interfaces; concrete implementations are unexported.

| Interface | Factory | Purpose |
|-----------|---------|---------|
| `Client` | `NewClient(baseURL, apiKey)` | Raw CRUD against the OneCLI API |
| `CredentialProvider` | `NewCredentialProvider(baseURL)` | Retrieves the `oc_` API key from `/api/user/api-key` |
| `SecretMapper` | `NewSecretMapper(registry)` | Converts `workspace.Secret` → `CreateSecretInput` |
| `SecretProvisioner` | `NewSecretProvisioner(client)` | Creates or updates secrets via `Client`, handles 409 conflicts |

## Client

`NewClient(baseURL, apiKey string) Client` — 30-second timeout, Bearer auth header.

### Secrets API

```go
// Create a secret; returns the created Secret or an *APIError.
secret, err := client.CreateSecret(ctx, onecli.CreateSecretInput{
    Name:        "github",
    Type:        "generic",
    Value:       "ghp_xxxx",
    HostPattern: "api\\.github\\.com",
    InjectionConfig: &onecli.InjectionConfig{
        HeaderName:  "Authorization",
        ValueFormat: "Bearer {value}",
    },
})

// Update an existing secret by ID (all fields optional).
err = client.UpdateSecret(ctx, secret.ID, onecli.UpdateSecretInput{
    Value: ptr("ghp_new"),
})

// List all secrets.
secrets, err := client.ListSecrets(ctx)

// Delete by ID.
err = client.DeleteSecret(ctx, secret.ID)
```

### Container Config

```go
cfg, err := client.GetContainerConfig(ctx)
// cfg.Env — map of proxy env vars to inject into the workspace container
// cfg.CACertificate — PEM-encoded CA cert
// cfg.CACertificateContainerPath — path where the cert should be written inside the container
```

### Networking Rules API

```go
rule, err := client.CreateRule(ctx, onecli.CreateRuleInput{
    Name:        "allow-github",
    HostPattern: "github\\.com",
    Action:      "allow",
    Enabled:     true,
})

rules, err := client.ListRules(ctx)
err = client.DeleteRule(ctx, rule.ID)
```

### Error Handling

Non-2xx responses return `*APIError{StatusCode int, Message string}`. Check with `errors.As`:

```go
var apiErr *onecli.APIError
if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
    // handle 409
}
```

## CredentialProvider

`NewCredentialProvider(baseURL string) CredentialProvider` — 10-second timeout, no auth required on the bootstrap call (local mode creates the user on first access).

```go
provider := onecli.NewCredentialProvider("http://localhost:8080")
apiKey, err := provider.APIKey(ctx)
// apiKey starts with "oc_"
```

## SecretMapper

`NewSecretMapper(registry secretservice.Registry) SecretMapper` — converts `workspace.Secret` values (from `workspace.json`) into `CreateSecretInput` values ready for the API.

```go
mapper := onecli.NewSecretMapper(secretServiceRegistry)
input, err := mapper.Map(secret)
```

### Mapping rules

- **Known type** (e.g. `github`): looks up the `SecretService` in the registry; uses its `HostPattern()`, `Path()`, `HeaderName()`, and `HeaderTemplate()` fields. The secret's `Name` field overrides the type name if set.
- **`other` type**: uses the secret's own `Hosts`, `Path`, `Header`, `HeaderTemplate` fields. Only one host is allowed — split multi-host secrets into separate entries.
- Template conversion: kdn uses `${value}`, OneCLI uses `{value}` — the mapper converts automatically.
- `HostPattern` is always `"*"` when `Hosts` is nil or empty.

## SecretProvisioner

`NewSecretProvisioner(client Client) SecretProvisioner` — idempotent: creates a secret or, on 409, finds it by name and patches it.

```go
provisioner := onecli.NewSecretProvisioner(client)
err := provisioner.ProvisionSecrets(ctx, []onecli.CreateSecretInput{input1, input2})
```

On conflict the provisioner calls `ListSecrets` to find the ID, then `UpdateSecret`. It returns an error if the named secret cannot be found after a 409.

## Integration: Podman Runtime

The Podman runtime is the primary consumer of this package. The flow during workspace creation and start is:

### Workspace creation (`pkg/runtime/podman/create.go` — `setupOnecli`)

1. `NewCredentialProvider(baseURL).APIKey(ctx)` — get the API key after OneCLI starts
2. `NewClient(baseURL, apiKey)` — create the client
3. `NewSecretProvisioner(client).ProvisionSecrets(ctx, secrets)` — push secrets
4. `client.GetContainerConfig(ctx)` — retrieve proxy env vars and CA cert to inject into the workspace container

### Workspace start (`pkg/runtime/podman/network.go` — `configureNetworking`)

1. `NewCredentialProvider(baseURL).APIKey(ctx)`
2. `NewClient(baseURL, apiKey)`
3. `client.ListRules(ctx)` + `client.DeleteRule(ctx, id)` — wipe stale rules (idempotency)
4. `client.CreateRule(ctx, ...)` for each allowed host → `action: "allow"`
5. `client.CreateRule(ctx, ...)` with `hostPattern: "*"` → `action: "block"` (catch-all)

### Secret flow from manager (`pkg/instances/manager.go`)

The instances manager converts workspace-level secrets before calling the runtime:

```go
mapper := onecli.NewSecretMapper(secretServiceRegistry)
for _, s := range workspaceConfig.Secrets {
    input, err := mapper.Map(s)
    // collect input.InjectionConfig env vars for the workspace environment
    // pass inputs to runtime.CreateParams.OnecliSecrets
}
```

## Testing

Use a `*httptest.Server` to serve fake API responses and pass its URL to `NewClient` or `NewCredentialProvider`. The `Client` interface makes it straightforward to inject a fake:

```go
type fakeClient struct{ ... }
func (f *fakeClient) CreateSecret(_ context.Context, input onecli.CreateSecretInput) (*onecli.Secret, error) { ... }
// implement remaining methods ...
var _ onecli.Client = (*fakeClient)(nil)

provisioner := onecli.NewSecretProvisioner(&fakeClient{})
```

For `SecretMapper` tests, use a `secretservice.NewRegistry()` and register a fake `SecretService` implementation, or use the real `secretservicesetup.RegisterAll()`.
