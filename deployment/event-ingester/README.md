# armada-event-ingester

![Version: 0.0.0-latest](https://img.shields.io/badge/Version-0.0.0--latest-informational?style=flat-square) ![AppVersion: 0.0.0-latest](https://img.shields.io/badge/AppVersion-0.0.0--latest-informational?style=flat-square)

A helm chart for Armada Event Ingester component

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| additionalLabels | object | `{}` |  |
| additionalVolumeMounts | list | `[]` |  |
| additionalVolumes | list | `[]` |  |
| applicationConfig.batchMessages | int | `10000` |  |
| applicationConfig.batchSize | int | `1048576` |  |
| applicationConfig.metricsPort | int | `9000` |  |
| applicationConfig.pulsar.authenticationEnabled | bool | `false` |  |
| customServiceAccount | string | `nil` |  |
| image.repository | string | `"gresearchdev/event-ingester-ingester"` |  |
| image.tag | string | `"0.0.0-latest"` |  |
| prometheus.enabled | bool | `false` |  |
| prometheus.labels | object | `{}` |  |
| prometheus.scrapeInterval | string | `"15s"` |  |
| prometheus.scrapeTimeout | string | `"10s"` |  |
| replicas | int | `1` |  |
| resources.limits.cpu | string | `"300m"` |  |
| resources.limits.memory | string | `"1Gi"` |  |
| resources.requests.cpu | string | `"200m"` |  |
| resources.requests.memory | string | `"512Mi"` |  |
| serviceAccount | string | `nil` |  |
| strategy.rollingUpdate.maxUnavailable | int | `1` |  |
| strategy.type | string | `"RollingUpdate"` |  |
| terminationGracePeriodSeconds | int | `30` |  |
| tolerations | list | `[]` | Tolerations |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
