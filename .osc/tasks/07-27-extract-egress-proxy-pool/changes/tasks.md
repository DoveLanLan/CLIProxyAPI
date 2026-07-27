# Tasks: standalone EgressProxyPool extraction

- [x] Create the standalone Go module, config loader, migrated pool runtime,
  tests, server entrypoint, Dockerfile, Compose project, and documentation.
- [x] Add an authenticated versioned control API with bounded bodies, redacted
  errors, revision-safe subscription mutations, and expiring probe leases.
- [x] Replace CLIProxyAPI's embedded pool/controller/subscription implementation
  with a remote client and executor test seam.
- [x] Simplify CLIProxyAPI configuration and deployment ownership while keeping
  its Management API compatibility facade.
- [x] Run formatting, focused and complete tests, required builds, focused race,
  Compose validation, Docker image build, vet, and secret/path review.
- [x] Record change summary, regression results, rollback notes, and quality gate.
