# Toolchains

This directory records toolchain inputs shared by component packages.

Use small, reviewable files for pinned inputs:

- `dotnet.json` for .NET SDK and runtime expectations
- `go.json` for Go version expectations
- `containers.json` for container base images
- `external-tools.json` for Caddy, curl, h2spec, or other binary tools

Toolchain pins are shared so that normal component additions do not create separate CI and release stacks.
