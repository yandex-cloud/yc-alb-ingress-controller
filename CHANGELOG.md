# Changelog


## v0.2.17 - January 31, 2025
### Fixed
* Remove wrong requirements in CRDs

## v0.2.16 - January 16, 2025
### Added
* Modify request headers annotations
* Autoscale annotations
* Load balancing config annotations

## v0.2.15
### Added
* Logging request-id on error

## v0.2.14
### Changed
* Prohibit installation in default namespace

## v0.2.13
### Fixed
* Helm Chart securityContext problems

## v0.2.12
### Changed
* Ingress to VirtualHosts annotation mapping logic

## v0.2.11
### Added
* Values with commas in modify-header-response-replace
### Changed
* Kubebuilder project refresh
### Fixed
* Finalizers removing order

## v0.2.10
### Changed
* More informative errors

## v0.2.9
### Added
* GrpcBackendGroup resource
* Run parameter: enableDefaultHealthChecks

## v0.2.8
### Added
* Kubernetes events for errors and successful reconciliations
### Changed
* Bumped Go version to 1.22

## v0.2.7
### Added
* Route allowed methods
* Ingress default backend
* Healthchecks with service annotation
