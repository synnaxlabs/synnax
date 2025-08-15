# Release Cadence

- We release new versions of our platform once or twice a month depending on the
  complexity of the improvements we're making.
- We release bug fixes and security patches to published versions throughought the 
  normal release cycle.
- All releases are guaranteed to be backwards compatible.
- We follow [semantic versioning](https://semver.org/) for our releases.
- We publish release notes for each release, including a summary of the changes,
  detailed descriptions of bug fixes and improvements, and a list of breaking changes
  along with migration instructions.
- All new releases are optional. Update notifications are sent to all users, but no
  updates are applied automatically.

# Quality Assurance Process

- Each release begins by selecting a release candidate, and freezing all new feature
  development.
- We run an automated suite of over 6500 unit and integration as a first step.
- We run a thorough manual QA process on our in-house HITL test environment to guarantee
  safe operation in environments that mimic real-world deployments. 
- We have NI, LabJack Hardware, PLCs, Modbus devices, custom serial devices, and more 
  systems that we thoroughly test against.
- Produce a software bill of materials (SBOM) that is available on request.

# Release Integrity

- Our organization is verified by Apple and Microsoft, and we sign all release binaries
  with certificates to guarantee authenticity and integrity.
- We use GPG keys on Linux to sign all binaries, and GPG keys are published alongside
  each release.
- We can all third party dependencies for vulnerabilities and security issues.

# Data Storage & Backups

- We exclusively store data on our customers infrastructure, and, by default, do not
  have any access to our customers data.
 - Synnax stores all of its data in a single, tightly controlled directory. 
 - Backing up all of our data is as simple as copying the directory to a secure
   location.
 - We do not provide any automated backup services, but integrate well with industry
   standard backup solutions such as [Restic](https://restic.net/) and
   [Borg](https://www.borgbackup.org/).
 - The Synnax team develops and maintains an internal tool for extracting and recovering
   remaining data in case of corruption and/or partial loss.

# Network Security

- Our platform does not communicate with any external services over the network, and is
  designed to be operated in air-gapped environments.
- Synnax communicates over TPC/IP and/or UDP on port a single, configurable network port
  (typically 9090). All communication is encrypted using TLS, and all requests 
  authentication.
- Our software supports automatic reload and rotation of TLS certificates, and uses 
  modern ciphers with TLS version 1.3 or higher.

# Audit Logging

- Synnax implements an optional logging system that can store access records for
  auditing purposes. For each individual request, Synnax records the accessing user, 
  relevant information about the access resource, and any relevant permissions errors
  that ocurred during the request.