# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed
- Update RabbitMQ version to 3.8.x for 1.0.x releases

## [0.11.1] - 2020-05-18
### Added
- Add the `DATA_UPDATER_PLANT_AMQP_DATA_QUEUE_TOTAL_COUNT` variable to support Astarte >= 0.11.1

### Fixed
- Fixed a situation where Housekeeping might never enter in ready state with low CPU allocations

## [0.11.0] - 2020-04-14
### Added
- AstarteVoyagerIngress now has two more options in `api`: `serveMetrics` and `serveMetricsToSubnet`, to
  give fine-grained control on who has access to `/metrics`
- Astarte has a new option `astarteSystemKeyspace`, which allows to specify the replication factor for the
  main `astarte` keyspace upon cluster initialization (#83)

### Fixed
- When checking whether an upgrade can be performed, do not deadlock in case the cluster wasn't green
  when performing the request

### Changed
- All `/metrics` endpoints are no longer exposed by default

## [0.11.0-rc.1] - 2020-03-26
### Fixed
- Allocate resources correctly in Components when non-explicit per-component requirements are given
- tests: Fix Limits/Requests for the installed Astarte resource
- cfssl_ca_secret: Destroy Job in case it failed to avoid deadlocks on the internal CA

## [0.11.0-rc.0] - 2020-02-26
### Added
- Kubernetes Event support

### Fixed
- Ensure that the Astarte CR status takes into account VerneMQ and CFSSL StatefulSets as well.
- Fixed all omitempty fields for AstarteVoyagerIngress
- Fixed SSL and Host directives for Dashboard
- Fixed relative Dashboard path when deploying on a dedicated Host
- Fixed Housekeeping Key Generation in new clusters
- Fixed potential bug in Upgrade by draining RabbitMQ queues before migrating Cassandra

### Changed
- Added new configuration fields to Dashboard to support new 0.11 config format
- Use distroless nonroot static image for the Operator container, and ditch ubi as base image

## [0.11.0-beta.2] - 2020-01-25
### Added
- Upgrade support
- Custom affinity for all clustered deployments

### Changed
- Rewrote the Operator entirely in Go
- Updated Operator SDK to 0.14.0

## [0.11.0-beta.1]
### Added
- Added support for multi-queue Data Updater Plant and VerneMQ plugin. When migrating from single-queue deployments,
the recommended procedure is this: scale VerneMQ to 0 replicas to allow the existing queue to be emptied. When it
is empty, replace Data Updater Plant with the new version and bring VerneMQ back up to start publishing on the new queues.
- Molecule-based CI
- Added Finalizers upon Astarte deletion

### Fixed
- Reconciliation policy now prevents the Operator from reconciling forever for no reason
- Fix deprecations in Playbook
- Allow correctly passing the certificate expiry to CFSSL

### Changed
- Updated Operator SDK to 0.12
- Updated Kubernetes minimum requirement to 1.14
- Updated RBAC APIs

### Removed
- Remove deprecated `/realm` path, use `/realmmanagement` to access Realm Management

## [0.10.2] - 2019-12-09

## [0.10.1] - 2019-10-02
### Added
- Added static IP support to load balancer in AVI
- Support to Let's Encrypt Staging in AVI Certificates
- Allow setting a different version for each Astarte component
- Allow setting custom images for individual Astarte components
- Support setting maxResultsLimit in AppEngine API
- Support fetching images from private registries
- Allow setting a custom image for RabbitMQ
- Support enabling additional plugins in RabbitMQ

### Changed
- Update default RabbitMQ version to 3.7.15

### Fixed
- Ensure Let's Encrypt http-01 challenge works in AVI
- Fixed typo which caused RabbitMQ pods to have memory limits identical to requests, even when explicitly set otherwise
- AVI: Fixed typo which caused the Playbook operator to crash when Dashboard host was not defined
- Use mqtts URI scheme in PAIRING_BROKER_URL
- Fixed an issue which resulted in an invalid resource when using a Cassandra custom volume definition
- Fixed an issue which prevented using a custom TLS secret
- Remove extra leading / in URL rewrites

## [0.10.0] - 2019-04-17
### Changed
- Change Houseekeeping API secrets naming

### Fixed
- Enhance resource auto-distribution in the operator
- Use correct domain in APIs

## [0.10.0-rc.0] - 2019-04-03
### Added
- First Astarte Kubernetes operator release.
