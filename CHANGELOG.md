# Changelog


## v0.2.23 - April 7, 2025
### Fixed
* Fix backend group creation order

## v0.2.22 - April 4, 2025
### Fixed
* Few network interfaces handling
* References to service with few ports

## v0.2.21 - March 20, 2025
### Fixed
* Fix support for nodes with multiple network interfaces

## v0.2.20 - February 27, 2025
### Added
* Add ru-central1-d to chart
### Changed
* Sync chart version with product version

## v0.2.18 - February 13, 2025
### Fixed
* Dont create empty and unused backend groups

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
