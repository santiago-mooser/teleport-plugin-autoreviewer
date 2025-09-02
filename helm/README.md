# Teleport Auto-reviewer Helm Chart

A Helm chart for deploying the Teleport Auto-reviewer service that automatically reviews and rejects access requests based on configurable rules. Now includes optional tbot integration for automated identity management using Teleport Machine ID.

## Features

- **Automated Access Request Review**: Configure rules to automatically review and reject access requests
- **Flexible Deployment**: Support for both manual identity file management and automated tbot identity management
- **Security Best Practices**: Pod security contexts, network policies, and RBAC
- **Production Ready**: Includes health checks, resource limits, and pod disruption budgets
- **Observability**: Built-in health endpoints and optional monitoring integration

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- A Teleport cluster (v13.0+)
- For tbot integration: Machine ID configured in your Teleport cluster

## Installation

### Option 1: Manual Identity File Management

```bash
# Create the identity file secret manually
kubectl create secret generic my-teleport-identity \
  --from-file=identity=/path/to/your/identity/file

# Install the chart
helm install my-autoreviewer oci://your-registry/teleport-plugin-request-autoreviewer \
  --set teleport.addr="your-teleport-cluster.com:443" \
  --set teleport.identityFile="$(base64 -i /path/to/your/identity/file)"
```

### Option 2: Automated Identity Management with tbot

```bash
# Install the chart with tbot enabled
helm install my-autoreviewer oci://your-registry/teleport-plugin-request-autoreviewer \
  --set teleport.addr="your-teleport-cluster.com:443" \
  --set tbot.enabled=true \
  --set tbot.machineId.token="your-machine-id-token"
```

## Configuration

### Basic Configuration

| Parameter                          | Description               | Default                           |
| ---------------------------------- | ------------------------- | --------------------------------- |
| `teleport.addr`                    | Teleport cluster address  | `"your-teleport-cluster.com:443"` |
| `teleport.reviewer`                | Name of the reviewer      | `"teleport-plugin-request-autoreviewer"`         |
| `teleport.identityRefreshInterval` | Identity refresh interval | `"1h"`                            |

### Manual Identity Configuration

| Parameter               | Description                          | Default |
| ----------------------- | ------------------------------------ | ------- |
| `teleport.identityFile` | Base64 encoded identity file content | `""`    |

### tbot Configuration

| Parameter                         | Description                                                                   | Default                                   |
| --------------------------------- | ----------------------------------------------------------------------------- | ----------------------------------------- |
| `tbot.enabled`                    | Enable tbot for automated identity management                                 | `false`                                   |
| `tbot.image.repository`           | tbot image repository                                                         | `"public.ecr.aws/gravitational/teleport"` |
| `tbot.image.tag`                  | tbot image tag                                                                | `"18.1.5"`                                |
| `tbot.machineId.token`            | Machine ID token for authentication                                           | `""`                                      |
| `tbot.machineId.addr`             | Teleport cluster address for tbot (defaults to teleport.addr)                 | `""`                                      |
| `tbot.output.secretName`          | Name of the secret to create (defaults to `<release-name>-teleport-identity`) | `""`                                      |
| `tbot.output.renewInterval`       | Certificate renewal interval                                                  | `"20m"`                                   |
| `tbot.output.certificateLifetime` | Certificate lifetime                                                          | `"1h"`                                    |
| `tbot.kubernetes.inCluster`       | Use in-cluster Kubernetes authentication                                      | `true`                                    |
| `tbot.kubernetes.clusterName`     | Kubernetes cluster name for cross-cluster auth                                | `""`                                      |

### Rejection Rules Configuration

Configure automatic rejection rules:

```yaml
rejection:
  defaultMessage: "Access request rejected due to policy violation"
  rules:
    - name: "Production Access Rule"
      roles_regex: "^prod-.*"
      reason_regex: "TICKET-\\d+"
      message: "Production access requires a valid ticket number"
```

### Security Configuration

| Parameter               | Description                | Default         |
| ----------------------- | -------------------------- | --------------- |
| `podSecurityContext`    | Pod security context       | See values.yaml |
| `securityContext`       | Container security context | See values.yaml |
| `networkPolicy.enabled` | Enable network policy      | `true`          |

### Resource Management

| Parameter                   | Description    | Default   |
| --------------------------- | -------------- | --------- |
| `resources.limits.cpu`      | CPU limit      | `"500m"`  |
| `resources.limits.memory`   | Memory limit   | `"512Mi"` |
| `resources.requests.cpu`    | CPU request    | `"100m"`  |
| `resources.requests.memory` | Memory request | `"128Mi"` |

## tbot Integration Details

When `tbot.enabled=true`:

1. **Separate tbot deployment** is created with Machine ID authentication
2. **Automated secret management** - tbot creates and maintains the identity secret
3. **RBAC permissions** - tbot gets necessary permissions to manage secrets
4. **Shared secret** - Both tbot and autoreviewer use the same secret name
5. **Kubernetes authentication** - tbot uses service account tokens for auth

### Machine ID Setup

Before using tbot integration, you need to:

1. Create a Machine ID in your Teleport cluster
2. Configure the Machine ID with appropriate roles
3. Generate a join token for Kubernetes authentication
4. Use the token in the `tbot.machineId.token` parameter

Example Machine ID configuration:
```bash
# Create Machine ID
tctl create --filename - <<EOF
kind: bot
version: v1
metadata:
  name: autoreviewer-bot
spec:
  roles: ["access-reviewer"]
EOF

# Create join token
tctl tokens add --type=bot --bot-name=autoreviewer-bot --format=text
```

## Monitoring

The chart includes health check endpoints and optional monitoring integration:

- Health endpoint: `/health`
- Metrics endpoint: `/metrics` (if enabled)

Enable monitoring:
```yaml
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
```

## Security Considerations

- **Network Policies**: Restrict traffic to necessary ports and protocols
- **Pod Security Standards**: Enforce restricted security standards
- **RBAC**: Minimal required permissions for both autoreviewer and tbot
- **Secret Management**: Identity files are stored in Kubernetes secrets with restricted access
- **Image Security**: Use specific image tags and consider image scanning

## Troubleshooting

### tbot Issues

1. **Check tbot logs**:
   ```bash
   kubectl logs deployment/my-autoreviewer-tbot
   ```

2. **Verify Machine ID token**:
   ```bash
   kubectl get secret my-autoreviewer-tbot-token -o yaml
   ```

3. **Check RBAC permissions**:
   ```bash
   kubectl auth can-i create secrets --as=system:serviceaccount:default:my-autoreviewer-tbot
   ```

### Identity Issues

1. **Check identity secret**:
   ```bash
   kubectl get secret my-release-teleport-identity -o yaml
   ```

2. **Verify secret mounting**:
   ```bash
   kubectl exec deployment/my-autoreviewer -- ls -la /etc/teleport/
   ```

## Migration from Manual to tbot

To migrate from manual identity management to tbot:

1. **Enable tbot** and set the Machine ID token
2. **Keep existing identity file** temporarily for rollback
3. **Verify tbot creates the secret** successfully
4. **Remove manual identity file** once confirmed working

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with `helm lint` and `helm template`
5. Submit a pull request

## License

This chart is licensed under the Apache 2.0 License.
