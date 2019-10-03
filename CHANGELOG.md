# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
