# gh-team

Extension for [gh](https://cli.github.com/) to show GitHub organization team hierarchy.

## Installation

```sh
gh extension install muniere/gh-team
```

## Usage

```
gh team <organization> [options]
```

### Arguments

| Argument         | Description                          |
| ---------------- | ------------------------------------ |
| `<organization>` | GitHub organization name (required)  |

### Options

| Option           | Description                                  |
| ---------------- | -------------------------------------------- |
| `--format=<type>` | Output format: `tree` (default) or `list`   |
| `--help`, `-h`   | Show help message                            |

## Examples

Display team structure as a tree:

```sh
gh team your-org
gh team your-org --format=tree
```

```
your-org
├── backend/
│   ├── api
│   └── infra
├── frontend/
│   └── web
└── platform

Total teams: 5
```

Display team structure as a list:

```sh
gh team your-org --format=list
```

```
your-org/backend
your-org/backend/api
your-org/backend/infra
your-org/frontend
your-org/frontend/web
your-org/platform

Total teams: 5
```

## Requirements

- [gh](https://cli.github.com/) with `read:org` scope

If you get a 404 error, refresh your auth token with the required scope:

```sh
gh auth refresh -h github.com -s read:org
```

## License

MIT
