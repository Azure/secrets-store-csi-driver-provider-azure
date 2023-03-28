# Release Management
> Note: This document is work in progress.

## Overview
This document describes **Azure Key Vault Provider for Secrets Store CSI Driver** project release management, which talks about versioning, branching and cadence.

## Legend

- **X.Y.Z** refers to the version (git tag) of AKV provider for Secrets Store CSI Driver that is released. This is the version of the AKV provider for Secrets Store CSI Driver image.
 
- **Milestone** should be designed to include feature sets to accommodate monthly release cycles including test gates. GitHub milestones are used by maintainers to manage each release. PRs and Issues for each release should be created as part of a corresponding milestone.

- **Test gates** should include soak tests and upgrade tests from the last minor version.


## Versioning
This project strictly follows [semantic versioning](https://semver.org/spec/v2.0.0.html). All releases will be of the form _vX.Y.Z_ where X is the major version, Y is the minor version and Z is the patch version. Current releases do not have version prefix *_`v`_*. Starting 0.1.0 we will add the version prefix, viz., _`v0.1.0`_

### Patch releases
- Patch releases provide users with bug fixes and security fixes. They do not contain new features.

### Minor releases
- Minor releases contain security and bug fixes as well as _**new features**_.

- They are backwards compatible.

### Major releases
- Major releases contain breaking changes. Breaking changes refer to schema changes and behavior changes of Secrets Store CSI Driver that may require a clean install during upgrade and it may introduce changes that could break backward compatibility.

- Ideally we will avoid making multiple major releases to be always backward compatible, unless project evolves in important new directions and such release is necessary.


## Release Cadence and Branching
- AKV provider for Secrets Store CSI Driver follows `monthly` release schedule.

- A new release would be created in _`second week`_ of each month. This schedule not only allows us to do bug fixes, but also provides an opportunity to address underlying image vulnerabilities if any.

- The release candidate (RC) images will be published from the master branch with tags. Once we validate RC image, we'll cut a release branch.

- The new version is decided as per above guideline and release branch should be created from `master` with name `release-<version>`. For eg. `release-v0.1.0`. Then build the image from release branch.

- Any `fixes` or `patches` should be merged to master and then `cherry pick` to the release branch.


## Acknowledgement

This document builds on the ideas and implementations of release processes from projects like [Gatekeeper](https://github.com/open-policy-agent/gatekeeper/blob/master/docs/Release_Management.md), [Helm](https://helm.sh/docs/topics/release_policy/#helm) and Kubernetes. 