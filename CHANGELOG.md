# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.10.1] - Unreleased
### Added
- Added static IP support to load balancer in AVI
- Support to Let's Encrypt Staging in AVI Certificates
- Allow setting a different version for each Astarte component

### Fixed
- Ensure Let's Encrypt http-01 challenge works in AVI
- Fixed typo which caused RabbitMQ pods to have memory limits identical to requests, even when explicitly set otherwise

## [0.10.0] - 2019-04-17
### Changed
- Change Houseekeeping API secrets naming

### Fixed
- Enhance resource auto-distribution in the operator
- Use correct domain in APIs

## [0.10.0-rc.0] - 2019-04-03
### Added
- First Astarte Kubernetes operator release.
