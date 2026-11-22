# Security Policy — OpenMimo

## Supported Versions
| Version | Supported |
| ------- | --------- |
| 1.x     | Yes       |

## Security Architecture
- **Local-first by default**: Projects (`.omimo`) are stored locally in the OS application data directory.
- **No telemetry or external tracking**: The only outbound network request is an optional, non-blocking check to `https://api.github.com/repos/aedmadrid/openmimo/releases/latest` for new releases (and optional user-initiated IPFS sharing or AI endpoint calls).
- **Host-controlled collaboration**: Remote peers can only modify a shared `.omimo` project while the host machine has the project actively open.
- **MCP local binding**: The HTTP MCP endpoint binds locally (`127.0.0.1`) and can be disabled at any time from the MCP status button.

## Reporting a Vulnerability
Please report security vulnerabilities privately via GitHub Security Advisories at:
https://github.com/aedmadrid/openmimo/security/advisories/new
