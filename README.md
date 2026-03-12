# raclette

Multi-repo toolkit for dept44 microservices. Format, verify, scan dependencies, swap artifacts, and update OpenAPI specs — all from a single TUI.

## Install

```bash
go install github.com/CheeziCrew/raclette@latest
```

Or grab a binary from [Releases](https://github.com/CheeziCrew/raclette/releases).

## Usage

```bash
raclette          # Launch the TUI from your service directory
```

Pick a command, select target repos, fill in any prompts, and let it rip. Maven commands run up to 3 builds in parallel.

### Commands

**Maven** — runs `mvn` across selected repos:

| Command | What it does |
|---------|-------------|
| Format All | `clean dept44-formatting:apply` |
| Verify All | `clean dept44-formatting:apply verify` |
| Update Own Specs | Run integration tests and overwrite your service's OpenAPI spec |

**Scan** — read-only analysis across repos:

| Command | What it does |
|---------|-------------|
| Find Dependency | Which repos use a specific dependency? |
| Find Stale Specs | List repos with outdated OpenAPI integration specs |

**Transform** — modifies files across repos:

| Command | What it does |
|---------|-------------|
| Replace Dependency Version | Bump a dependency version in pom.xml |
| Swap Dependency | Replace one artifact with another |
| Bump Parent Version | Update dept44-parent version |
| Update Integration Specs | Pull fresh specs from target services into your integrations |

## Prerequisites

| What | Required | Why |
|------|----------|-----|
| Maven | Yes | Maven commands need it |
| Java | Yes | dept44 services are Java |
| [Go 1.24+](https://go.dev/dl/) | Only for `go install` | Not needed if using a prebuilt binary |

## Acknowledgements

**Theo the Cat** — moral support department, head of naps division.

## License

[MIT](LICENSE)
