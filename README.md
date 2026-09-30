# CYPHONIC adapter

[![License](https://img.shields.io/badge/License-MIT-orange.svg)](https://opensource.org/licenses/MIT)

[![Adapterd](https://img.shields.io/badge/Adapterd-0.3.2-red.svg)](https://github.com/Pluslab/cyphonic-adapter)

[![Go](https://img.shields.io/badge/Go-1.25.5-blue.svg)](https://pkg.go.dev/golang.org/dl/go1.25.5)
[![Linux](https://img.shields.io/badge/Kernel-5.15.84-black.svg)](https://lkml.org/lkml/2021/10/31/203)

[![adapter-reviewdog](https://github.com/Pluslab/cyphonic-adapter/actions/workflows/adapter-reviewdog.yaml/badge.svg)](https://github.com/Pluslab/cyphonic-adapter/actions/workflows/adapter-reviewdog.yaml)

[![Go](https://github.com/Pluslab/cyphonic-adapter/actions/workflows/go.yaml/badge.svg)](https://github.com/Pluslab/cyphonic-adapter/actions/workflows/go.yaml)

## Cloning

### Requirements

| Language/FrameWork |  Version |
| :----------------- | -------: |
| go                 |   1.25.5 |
| Docker desktop     |   4.16.2 |
| Docker engine      | 20.10.22 |
| Compose            |   2.15.1 |

| Machine information |                        Version |
| :------------------ | -----------------------------: |
| Kernel version      |                    5.15.84-v8+ |
| Hardware            |                ARM64 (aarch64) |
| Host OS             | Debian GNU/Linux 11 (bullseye) |

### Downloading

```sh
$ git clone git@github.com:Pluslab/cyphonic-adapter.git
$ cd cyphonic-adapter
```

### Initial setting: plugins

```sh
### asdf: https://asdf-vm.com/guide/getting-started.html
$ brew update && brew install asdf

### plugins install
$ make plugin-install
```

### Initial setting: pre-commit

```sh
### pre-commit: https://pre-commit.com/
$ brew update && $ brew install pre-commit

### sets pre-commit
$ pre-commit install
```

## Supported Platform

### features

Under construction...

### platforms

- [Raspberry Pi4 Model B Cortex-A72(ARMv8) BCM2711 @1.5GHz 4cores 4threads 8GB DDR3](https://www.raspberrypi.com/products/raspberry-pi-4-model-b/)
- [OpenBlocks IX9 (debian) Intel(R) Atom(TM) CPU E3845 @1.91GHz 4cores 4threads 8GB DDR3](https://www.plathome.co.jp/product/openblocks/ix9-debian/)
- [Banana Pi R2 Quad-core ARM Cortex-A7 4cores 4threads 2GB DDR3](https://bananapi.gitbook.io/banana-pi-bpi-r2-open-source-smart-router/)

## Project managers

- [Issues](https://github.com/Pluslab/cyphonic-adapter/issues)
- [Pull Request](https://github.com/Pluslab/cyphonic-adapter/pulls)
- [Projects](https://github.com/Pluslab/cyphonic-adapter/projects/1)
- [Wiki](https://github.com/Pluslab/cyphonic-adapter/wiki)

## Contributing

Bug reports and pull requests are welcome on GitHub at [https://github.com/Pluslab/cyphonic-adapter](https://github.com/Pluslab/cyphonic-adapter).

This project is intended to be a safe, welcoming space for collaboration, and contributors are expected to adhere to the [Contributor Covenant](http://contributor-covenant.org) code of conduct.

### Code of Conduct

Everyone interacting in this project’s codebases, issue trackers, chat rooms and mailing lists is expected to follow the [code of conduct](./CODE_OF_CONDUCT.md).

### Thanks to the contributors of CYPHONIC Adapter!

<a href="https://github.com/GotoRen"><img src="https://avatars.githubusercontent.com/u/63791288?v=4" title="GotoRen" width="80" height="80"></a>
<a href="https://github.com/mitsu3s"><img src="https://avatars.githubusercontent.com/u/119998577?v=4" title="mitsu3s" width="80" height="80"></a>

## LICENSE

[MIT License](./LICENSE)
