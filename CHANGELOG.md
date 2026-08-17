# Changelog

## [1.8.7](https://github.com/batonogov/gitlab-auto-mr/compare/v1.8.6...v1.8.7) (2026-08-17)


### Maintenance

* **deps:** bump golang in the docker-dependencies group ([#141](https://github.com/batonogov/gitlab-auto-mr/issues/141)) ([b76a7ea](https://github.com/batonogov/gitlab-auto-mr/commit/b76a7eae82895299bc96289db14034a88d05aa01))
* **deps:** bump the github-actions group across 1 directory with 5 updates ([#139](https://github.com/batonogov/gitlab-auto-mr/issues/139)) ([ff715a8](https://github.com/batonogov/gitlab-auto-mr/commit/ff715a82c80ed4f33cd7f6e7a3cde7fd399e8392))
* **deps:** bump the github-actions group with 2 updates ([#142](https://github.com/batonogov/gitlab-auto-mr/issues/142)) ([e3e4cb2](https://github.com/batonogov/gitlab-auto-mr/commit/e3e4cb2480c310e7ad0704dd3e3a82502a2140df))

## [1.8.6](https://github.com/batonogov/gitlab-auto-mr/compare/v1.8.5...v1.8.6) (2026-07-13)


### Maintenance

* **deps:** bump golang in the docker-dependencies group ([#136](https://github.com/batonogov/gitlab-auto-mr/issues/136)) ([d9411f9](https://github.com/batonogov/gitlab-auto-mr/commit/d9411f9e94020cd4e125b381b0c5f2b6b0c8720f))
* **deps:** bump the github-actions group with 7 updates ([#135](https://github.com/batonogov/gitlab-auto-mr/issues/135)) ([8ca58fc](https://github.com/batonogov/gitlab-auto-mr/commit/8ca58fcb4521b185fb82c7feded4c3de667e0200))


### Continuous Integration

* pin GitHub Actions to commit SHAs ([#133](https://github.com/batonogov/gitlab-auto-mr/issues/133)) ([f05b58f](https://github.com/batonogov/gitlab-auto-mr/commit/f05b58f4cb3977563fb5b141783733e5b53acec6))

## [1.8.5](https://github.com/batonogov/gitlab-auto-mr/compare/v1.8.4...v1.8.5) (2026-06-29)


### Bug Fixes

* allow unsetting Squash/RemoveSourceBranch via --update-mr ([#66](https://github.com/batonogov/gitlab-auto-mr/issues/66)) ([b4b8a12](https://github.com/batonogov/gitlab-auto-mr/commit/b4b8a12d5e44b84d7e0ee416ac8f8ecc82024c0c))
* URL-encode branch names and add per_page=1 in getExistingMR ([#65](https://github.com/batonogov/gitlab-auto-mr/issues/65), [#70](https://github.com/batonogov/gitlab-auto-mr/issues/70)) ([76330c7](https://github.com/batonogov/gitlab-auto-mr/commit/76330c7b6076c02c3c8c116233306c340096793c))
* warn on getIssueData failure with --use-issue-name ([#67](https://github.com/batonogov/gitlab-auto-mr/issues/67)) ([e4077ae](https://github.com/batonogov/gitlab-auto-mr/commit/e4077ae15825573bac906258b575640d5089ac63))


### Maintenance

* **deps:** bump actions/cache from 5 to 6 in the github-actions group ([#132](https://github.com/batonogov/gitlab-auto-mr/issues/132)) ([85f43de](https://github.com/batonogov/gitlab-auto-mr/commit/85f43dee1c99ce6c937ff1f05bee4b1e00017eb7))
* **deps:** bump actions/checkout in the github-actions group ([#131](https://github.com/batonogov/gitlab-auto-mr/issues/131)) ([4fccd5d](https://github.com/batonogov/gitlab-auto-mr/commit/4fccd5db01403dc33eb9fc47cb0c6e92d098d080))
* **deps:** bump alpine in the docker-dependencies group ([#130](https://github.com/batonogov/gitlab-auto-mr/issues/130)) ([3108555](https://github.com/batonogov/gitlab-auto-mr/commit/31085554353a9de7d548e3130cc6e7bd5386b38a))
* fix lint findings (goconst, importShadow) ([1b3bbab](https://github.com/batonogov/gitlab-auto-mr/commit/1b3bbabeec92e74c0664031556a76da17391f58f))
* remove Python legacy .gitignore entries ([#111](https://github.com/batonogov/gitlab-auto-mr/issues/111)) ([4b88ca9](https://github.com/batonogov/gitlab-auto-mr/commit/4b88ca9aaf258dc50229cb78833bfd0ea03cbd7c))


### Code Refactoring

* make parseFlags testable by returning an error ([#68](https://github.com/batonogov/gitlab-auto-mr/issues/68)) ([da235f3](https://github.com/batonogov/gitlab-auto-mr/commit/da235f398af9a7ef6633fbf8f873aeeb073aa763))


### Continuous Integration

* remove auto-pr workflow ([#117](https://github.com/batonogov/gitlab-auto-mr/issues/117)) ([a19c5b0](https://github.com/batonogov/gitlab-auto-mr/commit/a19c5b01684e2f89a55466497dba4c083d8184fb))
* run CI on all PRs and fix greetings action inputs ([#118](https://github.com/batonogov/gitlab-auto-mr/issues/118)) ([93a39c9](https://github.com/batonogov/gitlab-auto-mr/commit/93a39c96fe293956cfecee8cb873c38a8df03e9d))

## [1.8.4](https://github.com/batonogov/gitlab-auto-mr/compare/v1.8.3...v1.8.4) (2026-06-15)


### Maintenance

* **deps:** bump the docker-dependencies group across 1 directory with 2 updates ([#104](https://github.com/batonogov/gitlab-auto-mr/issues/104)) ([20b49b3](https://github.com/batonogov/gitlab-auto-mr/commit/20b49b3ed44566046b27c20dc9f01659a2c6f54f))

## [1.8.3](https://github.com/batonogov/gitlab-auto-mr/compare/v1.8.2...v1.8.3) (2026-05-18)


### Maintenance

* **deps:** bump golang in the docker-dependencies group ([#101](https://github.com/batonogov/gitlab-auto-mr/issues/101)) ([7a0c458](https://github.com/batonogov/gitlab-auto-mr/commit/7a0c458d656825c3a5b29201e5d62ea515f4e0d6))
* **deps:** bump sigstore/cosign-installer in the github-actions group ([#102](https://github.com/batonogov/gitlab-auto-mr/issues/102)) ([8358934](https://github.com/batonogov/gitlab-auto-mr/commit/8358934735581640b1acc5395468986c2551e614))
* **deps:** bump the docker-dependencies group across 1 directory with 2 updates ([b47a636](https://github.com/batonogov/gitlab-auto-mr/commit/b47a636c25de791e50fc3af5f72e4d7cfeac5ead))
* **deps:** bump the github-actions group across 1 directory with 3 updates ([2e28d1b](https://github.com/batonogov/gitlab-auto-mr/commit/2e28d1b95a40b0700b2a408a314cfbc988dfa134))

## [1.8.2](https://github.com/batonogov/gitlab-auto-mr/compare/v1.8.1...v1.8.2) (2026-03-14)


### Bug Fixes

* address review feedback on lint config and code style ([55fa356](https://github.com/batonogov/gitlab-auto-mr/commit/55fa35664303a86610188c4ffefd54e478bcd514))


### Maintenance

* **deps:** bump golang in the docker-dependencies group ([690fb66](https://github.com/batonogov/gitlab-auto-mr/commit/690fb6604c67523ea165a4f2527444fb0af7d543))
* **deps:** bump the github-actions group with 5 updates ([8b10149](https://github.com/batonogov/gitlab-auto-mr/commit/8b10149ef7c4829b1ae3398c25321ec296ae49f9))
* migrate golangci-lint config to v2 and fix all lint issues ([e5bf749](https://github.com/batonogov/gitlab-auto-mr/commit/e5bf749e4b4472e0ffa7f36e044545e3bea10be0))
* migrate golangci-lint config to v2 and fix lint issues ([d1f52ce](https://github.com/batonogov/gitlab-auto-mr/commit/d1f52ce90c2e3cb0a6bb487990d158c11be4b1fc))

## [1.8.1](https://github.com/batonogov/gitlab-auto-mr/compare/v1.8.0...v1.8.1) (2026-03-05)


### Bug Fixes

* reject --auto-merge with draft commit prefix ([ae97594](https://github.com/batonogov/gitlab-auto-mr/commit/ae97594b74601ac895f5b10bee98c7dd46723602))


### Documentation

* remove hardcoded versions from README ([00e23e0](https://github.com/batonogov/gitlab-auto-mr/commit/00e23e0d2459109398eea1422740bd9ef43c9f53))

## [1.8.0](https://github.com/batonogov/gitlab-auto-mr/compare/v1.7.5...v1.8.0) (2026-03-05)


### Features

* add --auto-merge flag ([a27ca7a](https://github.com/batonogov/gitlab-auto-mr/commit/a27ca7a9da750c758d486e9b0e52f809f83c855e))
* add --auto-merge flag to enable merge when pipeline succeeds ([c99b573](https://github.com/batonogov/gitlab-auto-mr/commit/c99b573d928c792ecba746de5e46b4ebbb9223f1))


### Bug Fixes

* adjust message when MR exists with --auto-merge ([bbde1ee](https://github.com/batonogov/gitlab-auto-mr/commit/bbde1ee8ab510f54c7b5a2a68f3dc7cca388de6f))
* distinguish empty vs malformed JSON in createMR response ([ee61a50](https://github.com/batonogov/gitlab-auto-mr/commit/ee61a50965fe3511789528db3a1a1ecdb539e932))
* handle empty response body from createMR gracefully ([3571d5c](https://github.com/batonogov/gitlab-auto-mr/commit/3571d5c489973f5a77d8496b1550018f49c225c0))
* return error for malformed JSON in createMR response ([351014a](https://github.com/batonogov/gitlab-auto-mr/commit/351014a6220f8c4f3ee9956d012f515995c19a01))


### Tests

* add TestAcceptMR406 for unresolved discussions error ([632abee](https://github.com/batonogov/gitlab-auto-mr/commit/632abee22651e73267cb9d5e5d5398c7b2ba0fa0))

## [1.7.5](https://github.com/batonogov/gitlab-auto-mr/compare/v1.7.4...v1.7.5) (2026-03-04)


### Maintenance

* **deps:** bump the github-actions group across 1 directory with 2 updates ([#84](https://github.com/batonogov/gitlab-auto-mr/issues/84)) ([ce4da1f](https://github.com/batonogov/gitlab-auto-mr/commit/ce4da1ff47725c9f99ba8bcab4d7d322ecbf942a))

## [1.7.4](https://github.com/batonogov/gitlab-auto-mr/compare/v1.7.3...v1.7.4) (2026-02-28)


### Maintenance

* **deps:** bump golang in the docker-dependencies group ([#81](https://github.com/batonogov/gitlab-auto-mr/issues/81)) ([4d0dda8](https://github.com/batonogov/gitlab-auto-mr/commit/4d0dda8cd6aed502710dec9ca8b1d33e88e616dd))

## [1.7.3](https://github.com/batonogov/gitlab-auto-mr/compare/v1.7.2...v1.7.3) (2026-02-20)


### Maintenance

* **deps:** bump golang in the docker-dependencies group ([#63](https://github.com/batonogov/gitlab-auto-mr/issues/63)) ([c3f4672](https://github.com/batonogov/gitlab-auto-mr/commit/c3f46725225417316037ecd02773dcc65f638f40))

## [1.7.2](https://github.com/batonogov/gitlab-auto-mr/compare/v1.7.1...v1.7.2) (2026-02-02)


### Maintenance

* **deps:** bump the docker-dependencies group across 1 directory with 2 updates ([#59](https://github.com/batonogov/gitlab-auto-mr/issues/59)) ([7b96f1d](https://github.com/batonogov/gitlab-auto-mr/commit/7b96f1db8d1c465d70f1efe62d44ccf3af985f2d))

## [1.7.1](https://github.com/batonogov/gitlab-auto-mr/compare/v1.7.0...v1.7.1) (2025-12-22)


### Maintenance

* **deps:** bump actions/checkout in the github-actions group ([#51](https://github.com/batonogov/gitlab-auto-mr/issues/51)) ([c99dd76](https://github.com/batonogov/gitlab-auto-mr/commit/c99dd761b7bb14be1b02647ba1161b4200e65dc0))
* **deps:** bump alpine from 3.23.0 to 3.23.2 in the docker-dependencies group ([#55](https://github.com/batonogov/gitlab-auto-mr/issues/55)) ([5c39ca8](https://github.com/batonogov/gitlab-auto-mr/commit/5c39ca895aede4aa8bcb9e95909d54d1d7ae666c))
* **deps:** bump golang in the docker-dependencies group ([#49](https://github.com/batonogov/gitlab-auto-mr/issues/49)) ([9e2ed13](https://github.com/batonogov/gitlab-auto-mr/commit/9e2ed13b4983e04d07825f03d5c9d7ce298a42b0))
* **deps:** bump golangci/golangci-lint-action ([#50](https://github.com/batonogov/gitlab-auto-mr/issues/50)) ([f9b6a8e](https://github.com/batonogov/gitlab-auto-mr/commit/f9b6a8e5aa11e68aaf1b6712b2102d7ed9c21842))
* **deps:** bump the docker-dependencies group with 2 updates ([#52](https://github.com/batonogov/gitlab-auto-mr/issues/52)) ([55b1ba4](https://github.com/batonogov/gitlab-auto-mr/commit/55b1ba4abbd87e74b1063a79601ec9680578afb3))
* **deps:** bump the github-actions group with 2 updates ([#54](https://github.com/batonogov/gitlab-auto-mr/issues/54)) ([13128ed](https://github.com/batonogov/gitlab-auto-mr/commit/13128ed697fff28926a2a9bf7fb3006b077ceb3a))
* include chore in release-please sections ([#56](https://github.com/batonogov/gitlab-auto-mr/issues/56)) ([5785c60](https://github.com/batonogov/gitlab-auto-mr/commit/5785c603664c4d517fbd937bbae7f1c419fef9fc))

## [1.7.0](https://github.com/batonogov/gitlab-auto-mr/compare/v1.6.2...v1.7.0) (2025-11-05)


### Features

* add explicit --update-mr flag requirement ([#47](https://github.com/batonogov/gitlab-auto-mr/issues/47)) ([0248376](https://github.com/batonogov/gitlab-auto-mr/commit/02483764c5187c411a3afa8258b5d9b63748caf9))

## [1.6.2](https://github.com/batonogov/gitlab-auto-mr/compare/v1.6.1...v1.6.2) (2025-06-18)


### Bug Fixes

* resolve release-please commit parsing issues ([#28](https://github.com/batonogov/gitlab-auto-mr/issues/28)) ([65fb3bb](https://github.com/batonogov/gitlab-auto-mr/commit/65fb3bb67e552de0d3fd4ff2d58ae108292255a0))

## [1.6.1](https://github.com/batonogov/gitlab-auto-mr/compare/v1.6.0...v1.6.1) (2025-06-11)


### Documentation

* update version examples to use existing version 1.6.0 ([#24](https://github.com/batonogov/gitlab-auto-mr/issues/24)) ([cfa86a6](https://github.com/batonogov/gitlab-auto-mr/commit/cfa86a6de5505af7897fc84cefdaf1e1dfc55c21))

## [1.6.0](https://github.com/batonogov/gitlab-auto-mr/compare/v1.5.0...v1.6.0) (2025-06-11)


### Features

* add Docker image information to releases ([34c19f7](https://github.com/batonogov/gitlab-auto-mr/commit/34c19f72280325982825973ad22a3cfb64bb2b9d))
* add Docker image information to releases ([a2c1cb0](https://github.com/batonogov/gitlab-auto-mr/commit/a2c1cb0ccdd76abb7d99fcaa5fb6cd3114d55df7))

## [1.5.0](https://github.com/batonogov/gitlab-auto-mr/compare/v1.4.2...v1.5.0) (2025-06-11)


### Features

* add tag validation to release workflow ([394a81c](https://github.com/batonogov/gitlab-auto-mr/commit/394a81cc4cc09b1078c731e27620b654b48306c2))


### Bug Fixes

* add packages write permission to docker job in release workflow ([1130d2b](https://github.com/batonogov/gitlab-auto-mr/commit/1130d2b3ff50620e391dcd46916c6b15f0f9eaa3))
* add packages write permission to docker job in release workflow ([10790ca](https://github.com/batonogov/gitlab-auto-mr/commit/10790ca52a5d29cc71819c3bc49eaa5a8740d5ac))
* resolve release workflow issues ([8fa43ea](https://github.com/batonogov/gitlab-auto-mr/commit/8fa43eaa518f18c4d3581feab8ee096ceb8fab1f))
* resolve release-please permissions and configuration issues ([771f3fc](https://github.com/batonogov/gitlab-auto-mr/commit/771f3fc2f13eb1181d0230b6ed1a28013833316d))
* resolve release-please permissions and configuration issues ([7e9d06f](https://github.com/batonogov/gitlab-auto-mr/commit/7e9d06f58ea4562b7bbc19eb106e286c7d644d01))
* автоматическая сборка бинарников и Docker образов при релизе ([a9ad0d1](https://github.com/batonogov/gitlab-auto-mr/commit/a9ad0d1714251230685d1c55a42ca5cf7dbb88f7))

## [1.4.2](https://github.com/batonogov/gitlab-auto-mr/compare/v1.4.1...v1.4.2) (2025-06-11)


### Bug Fixes

* автоматическая сборка бинарников и Docker образов при релизе ([a9ad0d1](https://github.com/batonogov/gitlab-auto-mr/commit/a9ad0d1714251230685d1c55a42ca5cf7dbb88f7))

## [1.4.1](https://github.com/batonogov/gitlab-auto-mr/compare/v1.4.0...v1.4.1) (2025-06-10)


### Bug Fixes

* resolve release workflow issues ([8fa43ea](https://github.com/batonogov/gitlab-auto-mr/commit/8fa43eaa518f18c4d3581feab8ee096ceb8fab1f))

## [1.4.0](https://github.com/batonogov/gitlab-auto-mr/compare/v1.3.0...v1.4.0) (2025-06-10)


### Features

* add tag validation to release workflow ([394a81c](https://github.com/batonogov/gitlab-auto-mr/commit/394a81cc4cc09b1078c731e27620b654b48306c2))

## [1.3.0](https://github.com/batonogov/gitlab-auto-mr/compare/v1.2.0...v1.3.0) (2025-06-10)


### Features

* add release-please configuration files ([e0bb5fc](https://github.com/batonogov/gitlab-auto-mr/commit/e0bb5fcf71f4f2e73be31d6754cd528e5c6c8fd9))
* Complete Go implementation with zero dependencies ([c3a9bd3](https://github.com/batonogov/gitlab-auto-mr/commit/c3a9bd32b00b6033cb98fdc56d12cdab34c05cb5))


### Bug Fixes

* resolve release-please permissions and configuration issues ([771f3fc](https://github.com/batonogov/gitlab-auto-mr/commit/771f3fc2f13eb1181d0230b6ed1a28013833316d))
* resolve release-please permissions and configuration issues ([7e9d06f](https://github.com/batonogov/gitlab-auto-mr/commit/7e9d06f58ea4562b7bbc19eb106e286c7d644d01))
