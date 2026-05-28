# Changelog

## [2.5.1](https://github.com/chenasraf/tx/compare/v2.5.0...v2.5.1) (2026-05-28)


### Bug Fixes

* **tmux:** respect base-index and pane-base-index ([b12a90e](https://github.com/chenasraf/tx/commit/b12a90ec3c89cdd4589a4c3050fd9333ffc1445e))

## [2.5.0](https://github.com/chenasraf/tx/compare/v2.4.1...v2.5.0) (2026-05-28)


### Features

* **cli:** autocomplete project names for prj ([9739424](https://github.com/chenasraf/tx/commit/973942440f3c4984e203023910689a4584c5683b))

## [2.4.1](https://github.com/chenasraf/tx/compare/v2.4.0...v2.4.1) (2026-04-21)


### Bug Fixes

* use -B for background flag to prevent flags conflict ([627fde8](https://github.com/chenasraf/tx/commit/627fde8b01f685ff8ba0705fd2735b89fd92b356))

## [2.4.0](https://github.com/chenasraf/tx/compare/v2.3.0...v2.4.0) (2026-04-07)


### Features

* add background flag ([54fa59b](https://github.com/chenasraf/tx/commit/54fa59be464c226e9da0510b8d2a290023782d4a))

## [2.3.0](https://github.com/chenasraf/tx/compare/v2.2.0...v2.3.0) (2026-04-04)


### Features

* ls sessions as table ([7a9ea1b](https://github.com/chenasraf/tx/commit/7a9ea1b41d1fff87cbd160409453502b56dad29d))

## [2.2.0](https://github.com/chenasraf/tx/compare/v2.1.0...v2.2.0) (2026-03-24)


### Features

* allow pane size config ([d067a6e](https://github.com/chenasraf/tx/commit/d067a6e964f3ccd6c7e1cf322aa3997e943381c5))

## [2.1.0](https://github.com/chenasraf/tx/compare/v2.0.0...v2.1.0) (2026-03-23)


### Features

* accept multiple session names for rm/kill ([296e135](https://github.com/chenasraf/tx/commit/296e13549e2e962c4b2287f214ca03ecc9359d85))

## [2.0.0](https://github.com/chenasraf/tx/compare/v1.5.0...v2.0.0) (2026-03-20)


### ⚠ BREAKING CHANGES

* tmux_local.yaml is no longer auto-discovered. Run `tx migrate` to add it as an explicit include. The `--local`/`-l` flag is removed in favor of `--config`/`-c`.

### Features

* replace tmux_local with config file includes ([2e801cd](https://github.com/chenasraf/tx/commit/2e801cd91e49d2547731a01fc97a48804aa3e7b2))

## [1.5.0](https://github.com/chenasraf/tx/compare/v1.4.1...v1.5.0) (2026-03-18)


### Features

* initial_window option ([3f82e19](https://github.com/chenasraf/tx/commit/3f82e194bb3352d7a8b889c2d190a118a0d0dd60))

## [1.4.1](https://github.com/chenasraf/tx/compare/v1.4.0...v1.4.1) (2026-02-09)


### Bug Fixes

* pgup/pgdown scroll ([1b80378](https://github.com/chenasraf/tx/commit/1b803786b82ffecf8cd0691ea4605dac8246ed68))

## [1.4.0](https://github.com/chenasraf/tx/compare/v1.3.0...v1.4.0) (2026-02-09)


### Features

* add support for aliases ([7534382](https://github.com/chenasraf/tx/commit/7534382360f8b02ebadf575c53ed55dbd14e1047))
* new UI ([cf1018a](https://github.com/chenasraf/tx/commit/cf1018abc3ff108f5f7802ea7ef9fa881955e24f))

## [1.3.0](https://github.com/chenasraf/tx/compare/v1.2.1...v1.3.0) (2026-02-08)


### Features

* config name case insensitivity ([5c71d7b](https://github.com/chenasraf/tx/commit/5c71d7bc11ace64f7d96be07fba0fa7e3dae843f))

## [1.2.1](https://github.com/chenasraf/tx/compare/v1.2.0...v1.2.1) (2026-02-01)


### Bug Fixes

* select pane with window name ([916bbef](https://github.com/chenasraf/tx/commit/916bbef6e6680a4719af24dc08d89e81e85f1bae))
* version/verbose flag ([e917d6a](https://github.com/chenasraf/tx/commit/e917d6a5fc1325b3f54561b3589a9194db8c08dd))

## [1.2.0](https://github.com/chenasraf/tx/compare/v1.1.0...v1.2.0) (2026-01-30)


### Features

* add kill command ([0c73e4c](https://github.com/chenasraf/tx/commit/0c73e4c502bea7ec1002bcdc94e2f10d479d2c30))
* add named layouts ([ffd722b](https://github.com/chenasraf/tx/commit/ffd722b6c90dfa902f9e46419e523edc51737336))
* add session autocompletions ([2e74146](https://github.com/chenasraf/tx/commit/2e741464c117c68e164dbe776a18557a5f8ddf22))

## [1.1.0](https://github.com/chenasraf/tx/compare/v1.0.1...v1.1.0) (2026-01-29)


### Features

* add version flag ([d913ae8](https://github.com/chenasraf/tx/commit/d913ae80f9be09d4fdb20e3cfd23df071abf1fe6))
* allow overriding default layout ([302ecb6](https://github.com/chenasraf/tx/commit/302ecb697ef4e0fad79a0e87af108646a24eb870))
* update possible config locations ([b52ef75](https://github.com/chenasraf/tx/commit/b52ef759b6d1cd92c76e903852d16d64430ee649))

## [1.0.1](https://github.com/chenasraf/tx/compare/v1.0.0...v1.0.1) (2026-01-29)


### Bug Fixes

* error & cancel outputs ([ae6de6b](https://github.com/chenasraf/tx/commit/ae6de6bf578bd059e88633d87cd45c09bbd27fdd))

## 1.0.0 (2026-01-29)


### Features

* initial commit ([0f3d2a3](https://github.com/chenasraf/tx/commit/0f3d2a36d95e6b01b425d0988309266da178177a))
