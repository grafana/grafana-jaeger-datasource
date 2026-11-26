# Contributing

## Signed commits are required

> [!IMPORTANT]
> All commits must be [signed](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits) (GPG, SSH, or S/MIME) to be merged into this repository. Pull requests with unsigned commits will need to be re-committed with signatures before they can be merged.

Thank you for your interest in contributing to the Jaeger data source for Grafana! We welcome contributions from the community.

Feel free to [browse open issues](https://github.com/grafana/grafana-jaeger-datasource/issues) or open a new one. For more general guidance, see [Grafana's Contributing Guide](https://github.com/grafana/grafana/blob/main/CONTRIBUTING.md).

This project adheres to the [Grafana Code of Conduct](https://github.com/grafana/grafana/blob/main/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Prerequisites

- [Git](https://git-scm.com/)
- [Go](https://golang.org/dl/) (see [go.mod](go.mod) for the minimum required version)
- [Mage](https://magefile.org/)
- [Node.js LTS](https://nodejs.org)
- [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm) (see [package.json](package.json) for the minimum required version)
- [Docker](https://docs.docker.com/get-docker/)

## Frontend

1. Install dependencies:

   ```shell
   npm install
   ```

2. Build plugin in development mode and watch for changes:

   ```shell
   npm run dev
   ```

3. Build plugin in production mode:

   ```shell
   npm run build
   ```

4. Run frontend tests:

   ```shell
   npm run test:ci
   ```

## Backend

1. Build the backend binaries:

   ```shell
   mage -v
   ```

2. Run backend tests:

   ```shell
   go test ./pkg/...
   ```

## Data Source Configuration Schema

[`pkg/schema/dsconfig.json`](pkg/schema/dsconfig.json) is the single source of truth for this data source's configuration (`root`, `jsonData` and `secureJsonData`). Provisioning, docs and automation consume it. See the [dsconfig README](https://github.com/grafana/dsconfig/tree/main/dsconfig), the [property reference](https://github.com/grafana/dsconfig/blob/main/dsconfig/schema.md) and the [JSON Schema](https://github.com/grafana/dsconfig/blob/main/dsconfig/schema.json).

The `$schema` URL is pinned to a dsconfig release. It must always match the `github.com/grafana/dsconfig/dsconfig` version in `go.mod`; bump them together.

### Layout

| File | Purpose |
| --- | --- |
| `pkg/schema/dsconfig.json` | The schema. Edit this. |
| `pkg/schema/dsconfig_test.go` | Conformance wiring, `SecureKeys` and the test-only settings model. |
| `pkg/schema/*.gen.json` | Generated. Never hand-edit. The build copies them into `dist/schema/`. |

### Adding a new settings option

1. Declare the field in `dsconfig.json` under `fields` and add its id to `groups[].fieldRefs`. Ids follow `<target>_<key>`.
2. Add the matching field, with a json tag equal to the schema `key`, to the settings model in `dsconfig_test.go`. Parity is enforced both ways. Secrets are the exception: add the key to `SecureKeys` instead.
3. Run `go generate ./pkg/schema/...` and commit the regenerated `.gen.json` files.
4. Run `go test ./pkg/schema/...`.

### When the conformance suite fails

- `SchemaArtifactInSync`: the `.gen.json` files are stale. Run `go generate ./pkg/schema/...`.
- `JSONDataMatchesStruct` / `JSONDataTypesMatchStruct`: a `jsonData` key or its type differs between the schema and the settings model. Fix whichever side is wrong.
- `SecureValuesMatchLoadSettings`: a `secureJsonData` key is missing from `SecureKeys` or the schema. Make them match.

## Local development environment

`npm run server` starts a local Jaeger instance and a Grafana instance with the plugin pre-provisioned:

```shell
npm run server
```

<!-- Add any plugin-specific environment variables or version-pinning notes here. -->

## E2E tests

```shell
npm run server
npm run e2e
```

Or, to install Playwright browsers first:

```shell
npx playwright install --with-deps
npm run server
npm run e2e
```

## Checklist

- If you touched data source settings, run `go generate ./pkg/schema/...` and commit the regenerated `.gen.json` artifacts.

## Release

You need commit access to the repository to publish a release.

1. Update the version number in `package.json`.
2. Update `CHANGELOG.md` with the changes included in the release.
3. Open a PR with the changes and merge it.
4. Follow the release process described [here](https://enghub.grafana-ops.net/docs/default/component/grafana-plugins-platform/plugins-ci-github-actions/010-plugins-ci-github-actions/#cd_1).
