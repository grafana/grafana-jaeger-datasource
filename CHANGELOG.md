# Changelog

## Unreleased

- Fix a connection freeze between Grafana and the plugin under heavy query load by updating google.golang.org/grpc to v1.85.0 ([grpc/grpc-go#9330](https://github.com/grpc/grpc-go/issues/9330))
- Fix security vulnerabilities (GHSA-cg2g-xh64-v4q7, GHSA-mrv2-3pqr-j5jr, CVE-2026-93749, CVE-2023-54404)

## 13.0.7

- Chore: Update dependencies

## 13.0.6

- Chore: Pin golangci-lint to 2.15.0 in development dependencies and CI/CD pipelines

## 13.0.5

- Fix security vulnerabilities (CVE-2026-102276, CVE-2026-102278, CVE-2026-102990)

## 13.0.4

- Fix security vulnerabilities (CVE-2026-84445)

## 13.0.3

- Chore: Update dependencies

## 13.0.2

- Bump go v1.26.7 and grafana-plugin-sdk-go v0.296.4 ([#70](https://github.com/grafana/grafana-jaeger-datasource/pull/70))
- Dependency updates ([#72](https://github.com/grafana/grafana-jaeger-datasource/pull/72))

## 13.0.1

- Sync PRs from grafana/grafana and bump frontend dependencies ([#49](https://github.com/grafana/grafana-jaeger-datasource/pull/49))

## 13.0.0

- Initial release.
