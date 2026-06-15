# Security Policy

ProtocolLab Components contains component wrappers, package manifests,
packaging scripts, and toolchain metadata. Report suspected security issues
privately.

## Reporting

- Do not open a public issue or PR for a suspected vulnerability.
- Use GitHub private vulnerability reporting if it is enabled for this
  repository.
- If GitHub private vulnerability reporting is unavailable, contact
  `security@incursa.com`.
- For general open-source or governance questions, contact `oss@incursa.com`.

Please include:

- affected file, commit, package, or branch
- reproduction steps
- observed impact
- whether the issue exposes secrets, private paths, credentials, unsafe
  defaults, or unsafe execution guidance

## Scope

Security concerns here are usually about leaked paths, credentials, unsafe
wrapper defaults, unsafe publication guidance, vulnerable build inputs,
generated artifacts that should not be committed, or docs that overstate the
public component boundary.

Component wrappers may invoke local runtimes such as .NET, Go, Bash, PowerShell,
Docker, Caddy, or nginx. Treat changes to execution entrypoints, dependency
requirements, and artifact packaging as security-relevant review areas.

## Response

We will triage privately and coordinate remediation before any public
disclosure when possible.
