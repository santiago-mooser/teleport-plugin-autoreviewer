# Teleport Auto-reviewer Helm Chart

A Helm chart for deploying the Teleport Auto-reviewer service that automatically reviews and rejects access requests based on configurable rules.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- Valid Teleport identity file
- Access to your container registry

## Installation

### 1. Build and Push Container Image

```bash
# Build the Docker image
docker build -t your-registry/teleport-autoreviewer:latest .

# Push to your container registry
docker push your-registry/teleport-autoreviewer:latest
```

### 2. Prepare Values File

Create a `values-prod.yaml` file with your configuration:

```yaml
image:
  repository: your-registry/teleport-autoreviewer
  tag: latest

teleport:
  addr: "your-teleport-cluster.com:443"
  identityFile: |
    -----BEGIN CERTIFICATE-----
    # Your Teleport identity certificate content
    -----END CERTIFICATE-----
    -----BEGIN PRIVATE KEY-----
    # Your Teleport private key content
    -----END PRIVATE KEY-----

rejection:
  rules:
    - name: "Production Access Rule"
      roles_regex: "^prod-(.*)$"
      reason_regex: "((.*)\\w+JIRA-\\d+(.*))"
      message: "Production access requires a valid JIRA ticket"

    - name: "Development Access Rule"
      roles_regex: "^dev-(.*)$"
      reason_regex: "((.*)\\w+DEV-\\d+(.*))"
      message: "Development access requires a valid DEV ticket"

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 256Mi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
```

### 3. Install the Chart

```bash
# Add the chart repository (if using a chart repository)
helm repo add your-repo https://your-chart-repo.com
helm repo update

# Install with custom values
helm install teleport-autoreviewer your-repo/teleport-autoreviewer \
  --namespace teleport-system \
  --create-namespace \
  -f values-prod.yaml

# Or install from local chart directory
helm install teleport-autoreviewer ./helm/teleport-autoreviewer \
  --namespace teleport-system \
  --create-namespace \
  -f values-prod.yaml
```

## Configuration

### Key Configuration Options

| Parameter                     | Description                    | Default                         |
| ----------------------------- | ------------------------------ | ------------------------------- |
| `image.repository`            | Container image repository     | `teleport-autoreviewer`         |
| `image.tag`                   | Container image tag            | `""` (uses Chart.appVersion)    |
| `teleport.addr`               | Teleport cluster address       | `your-teleport-cluster.com:443` |
| `teleport.identityFile`       | Teleport identity file content | `""`                            |
| `rejection.rules`             | List of rejection rules        | `[]`                            |
| `resources.limits.memory`     | Memory limit                   | `512Mi`                         |
| `resources.requests.cpu`      | CPU request                    | `100m`                          |
| `autoscaling.enabled`         | Enable HPA                     | `false`                         |
| `networkPolicy.enabled`       | Enable NetworkPolicy           | `true`                          |
| `podDisruptionBudget.enabled` | Enable PDB                     | `true`                          |

### Rejection Rules Configuration

Each rule supports the following fields:

- `name`: Human-readable rule name
- `roles_regex`: (Optional) Regex pattern for roles this rule applies to
- `reason_regex`: Regex pattern that access request reasons must match
- `message`: Custom rejection message

#### Rule Logic

1. **Role Filter**: If `roles_regex` is specified, the rule only applies to requests containing roles that match the pattern
2. **Reason Check**: If the rule applies, requests with reasons that DON'T match `reason_regex` are rejected

## Security Features

This chart implements multiple security best practices:

### Container Security
- Distroless base image with minimal attack surface
- Non-root user execution (UID 65532)
- Read-only root filesystem
- Dropped capabilities
- Security contexts with seccomp profiles

### Kubernetes Security
- Pod Security Standards compliance (restricted)
- NetworkPolicy for network segmentation
- RBAC with least privilege principles
- Service Account with minimal permissions
- Pod Disruption Budget for availability

### Resource Management
- Resource requests and limits
- Horizontal Pod Autoscaler support
- Anti-affinity rules for high availability

## Monitoring and Observability

### Health Checks
- Liveness probe on `/health` endpoint
- Readiness probe for traffic routing
- Configurable probe parameters

### Logging
- Structured JSON logging
- Request processing logs with detailed information
- Rule evaluation logs for debugging

## Troubleshooting

### Common Issues

1. **Identity File Issues**
   ```bash
   # Check if secret is created correctly
   kubectl get secret -n teleport-system
   kubectl describe secret teleport-autoreviewer-identity -n teleport-system
   ```

2. **Connection Issues**
   ```bash
   # Check pod logs
   kubectl logs -n teleport-system deployment/teleport-autoreviewer

   # Check network connectivity
   kubectl exec -n teleport-system deployment/teleport-autoreviewer -- nslookup your-teleport-cluster.com
   ```

3. **Rule Configuration Issues**
   ```bash
   # Check config map
   kubectl get configmap teleport-autoreviewer-config -n teleport-system -o yaml

   # Test regex patterns
   kubectl exec -n teleport-system deployment/teleport-autoreviewer -- cat /app/config.yaml
   ```

### Debug Mode

Enable debug logging by setting environment variables:

```yaml
env:
  - name: LOG_LEVEL
    value: "debug"
```

## Upgrading

```bash
# Upgrade to new version
helm upgrade teleport-autoreviewer your-repo/teleport-autoreviewer \
  --namespace teleport-system \
  -f values-prod.yaml

# Rollback if needed
helm rollback teleport-autoreviewer 1 --namespace teleport-system
```

## Uninstalling

```bash
helm uninstall teleport-autoreviewer --namespace teleport-system
```

## Development

### Local Testing

```bash
# Lint the chart
helm lint ./helm/teleport-autoreviewer

# Template and review output
helm template teleport-autoreviewer ./helm/teleport-autoreviewer \
  -f values-prod.yaml

# Dry run installation
helm install teleport-autoreviewer ./helm/teleport-autoreviewer \
  --namespace teleport-system \
  --create-namespace \
  -f values-prod.yaml \
  --dry-run
```

### Chart Testing

```bash
# Install test values
helm install test-release ./helm/teleport-autoreviewer \
  --namespace test \
  --create-namespace \
  --set image.tag=test

# Run tests
helm test test-release --namespace test
```

## Support

For issues and questions:
- Check the troubleshooting section above
- Review pod logs and events
- Consult Teleport documentation for identity file setup
- Open an issue in the project repository
