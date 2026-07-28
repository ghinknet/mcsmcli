# mcsmcli

A Go CLI tool for managing MCSManager (MCSM) panels, built against the official API documentation. Covers every documented endpoint: authentication persistence, panel overview, daemon management, full instance lifecycle (start/stop/restart/create/delete/command/logs/reinstall), user management, file management (including upload/download), and Docker image management.

([中文文档](README_zh.md))

## Install

```bash
go build -o mcsmcli .
```

Dependencies: `go.gh.ink/json`, `go.gh.ink/timex`, `go.gh.ink/toolbox` (GoVanityPath — fetched automatically). CLI framework: `github.com/spf13/cobra`.

## Authentication

Generate an API key in the panel user center (permissions match your account; treat admin keys with care).

```bash
# Save credentials (defaults to verifying connectivity via /api/overview; --no-verify to skip)
mcsmcli login --url https://panel.example.com --apikey <API_KEY> [--daemon <daemonId>]

mcsmcli whoami          # Show current profile and test connectivity
mcsmcli logout          # Delete saved credentials
```

Configuration is stored at `~/.config/mcsmcli/config.json` (0600 permissions; override with `MCSM_CONFIG`).

Multi-panel support:

```bash
mcsmcli --profile prod login --url ... --apikey ...
mcsmcli profile list
mcsmcli profile use prod
mcsmcli profile set-daemon <daemonId>   # Set a default daemon, then omit -d later
```

Environment variables `MCSM_URL` / `MCSM_APIKEY` / `MCSM_DAEMON` can temporarily override the config file; the global flags `--url` / `--apikey` / `-d` take the highest priority.

## Common Commands

```bash
# Overview & daemons
mcsmcli overview
mcsmcli daemon list
mcsmcli daemon add --ip 10.0.0.16 --port 24444 --key <daemonKey> --remarks "My Node"
mcsmcli daemon link <daemonId>
mcsmcli daemon update <daemonId> --remarks "New name" --available true
mcsmcli daemon delete <daemonId>

# Instances (-d can be omitted when a default daemon is set)
mcsmcli instance list -d <daemonId> [--name keyword]
mcsmcli instance info <uuid>
mcsmcli instance start|stop|restart|kill <uuid>
mcsmcli instance batch-start uuid1 uuid2@otherDaemonId
mcsmcli instance cmd <uuid> say hello
mcsmcli instance log <uuid> --size 64
mcsmcli instance create --file config.json      # InstanceConfig JSON, - for stdin
mcsmcli instance update <uuid> --nickname "New name"
mcsmcli instance delete <uuid> [--files]        # --files also deletes instance files
mcsmcli instance upgrade <uuid>                 # Triggers the update command
mcsmcli instance reinstall <uuid> --target-url https://.../server.zip

# Users
mcsmcli user list [--role 10]
mcsmcli user create --name bob --password '...' --permission 1
mcsmcli user update <uuid> --permission 10      # or --file config.json for full config
mcsmcli user delete <uuid>...

# Files (first argument is the instance uuid)
mcsmcli file ls <uuid> /
mcsmcli file cat <uuid> /server.properties
mcsmcli file write <uuid> /eula.txt --text 'eula=true'
mcsmcli file download <uuid> /backup/world.zip ./world.zip
mcsmcli file upload <uuid> ./plugin.jar /plugins
mcsmcli file cp|mv <uuid> <src> <dst>
mcsmcli file zip <uuid> /backup.zip /world /config
mcsmcli file unzip <uuid> /backup.zip /restore --code utf-8
mcsmcli file rm|touch|mkdir <uuid> <path>

# Docker
mcsmcli image list|containers|networks
mcsmcli image build --name mcsm-custom --tag latest --dockerfile ./Dockerfile
mcsmcli image progress
```

Add `--json` to any query command to output the panel's raw data for scripting.

## Notes

- File upload/download use a two-stage process as documented: first request one-time credentials from the panel, then transfer directly with the daemon. The direct-connection protocol defaults to matching the panel (https panel → https daemon); use `MCSM_DAEMON_SCHEME=http|https` to override.
- Large file transfers are not constrained by `--timeout` (default 30s, applies to regular API requests only).
- `daemon update` fetches the current daemon config first and merges your specified fields; unspecified fields keep their current values (the daemon's apiKey is not returned by the panel — pass `--key` explicitly if you need to preserve it).
