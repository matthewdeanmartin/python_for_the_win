# Security policy

Report suspected vulnerabilities privately using the repository's GitHub Security
tab and **Report a vulnerability**, when available. If private reporting is not
enabled, ask the maintainer to enable it without posting exploit details, secrets,
or personal logs in a public issue. There is no guaranteed response SLA.

Include the pftw version, Windows version, affected command, impact, and a minimal
reproducer with synthetic data. Remove credentials, private URLs, and personal
paths. Do not test against other people's systems.

Security fixes target the latest release and the development branch; older
releases have no promised backports. pftw manages executable Python runtimes; it
does not sandbox scripts or make third-party Python packages trustworthy.

See [security architecture and roadmap](docs/security.md) for trust boundaries,
existing defenses, remaining limitations, and maintainer controls.
