classzz
====
[![Build Status](https://travis-ci.org/bourbaki-czz/classzz.png?branch=master)](https://travis-ci.org/bourbaki-czz/classzz)
[![Go Report Card](https://goreportcard.com/badge/github.com/classzz/classzz)](https://goreportcard.com/report/github.com/classzz/classzz)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/classzz/classzz)

classzz is an alternative full node bitcoin cash implementation written in Go (golang).

This project is a port of the [btcd](https://github.com/btcsuite/btcd) codebase to Bitcoin Cash. It provides a high powered
and reliable blockchain server which makes it a suitable backend to serve blockchain data to lite clients and block explorers
or to power your local wallet.

classzz does not include any wallet functionality by design as it makes the codebase more modular and easy to maintain. 
The [czzwallet](https://github.com/classzz/czzwallet) is a separate application that provides a secure Bitcoin Cash wallet 
that communicates with your running classzz instance via the API.

## Table of Contents

- [Requirements](#requirements)
- [Install](#install)
  - [Install prebuilt packages](#install-pre-built-packages)
  - [Build from Source](#build-from-source)
- [Getting Started](#getting-started)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## Requirements

[Go](http://golang.org) 1.9 or newer.

## Install

### Install Pre-built Packages

The easiest way to run the server is to download a pre-built binary. You can find binaries of our latest release for each operating system at the [releases page](https://github.com/classzz/classzz/releases).

### Build from Source

If you prefer to install from source do the following:

- Install Go according to the installation instructions here:
  http://golang.org/doc/install

- Run the following commands to obtain btcd, all dependencies, and install it:

```bash
go get github.com/classzz/classzz
```

This will download and compile `classzz` and put it in your path.

If you are a classzz contributor and would like to change the default config file (`classzz.conf`), make any changes to `sample-classzz.conf` and then run the following commands:

```bash
go-bindata sample-classzz.conf  # requires github.com/go-bindata/go-bindata/
gofmt -s -w bindata.go
```

## Getting Started
The V2.0 main network needs to download dogecoin and litecoin nodes and enable RPC services.

Dogecoin node configuration example:

```text
server = 1
rpcuser =root # RPC user name
rpcpassword = admin # RPC password
rpcallowip =127.0.0.1 # need to enable all access to 0.0.0.0/0
rpcbind = 0.0.0.0
rpcport = 9999 # RPC ports
txindex = 1
```
Litecoin node configuration example:
```text
server=1
rpcuser=root # RPC user name
rpcpassword=admin # RPC password
rpcallowip=127.0.0.1 # need to enable all access to 0.0.0.0/0
rpcbind=0.0.0.0
rpcport=19200 # RPC ports
txindex=1
```
To start classzz with default options just run:

```bash
./czzd
```

You'll find a large number of runtime options with the help flag. All of them can also be set in a config file.
See the [sample config file](https://github.com/classzz/classzz/blob/master/sample-classzz.conf) for an example of how to use it.

```bash
./czzd --help
```

You can use the common json RPC interface through the `czzctl` command:

```bash
./czzctl --help

./czzctl --listcommands
```

Classzz separates the node and the wallet. Commands for the wallet will work when you are also running
[czzwallet](https://github.com/classzz/czzwallet):

```bash
./czzctl -u username -P password --wallet getnewaddress
```

## Documentation

The documentation is a work-in-progress.  It is located in the [docs](https://github.com/classzz/classzz/tree/master/docs) folder.

## Contributing

Contributions are definitely welcome! Please read the contributing [guidelines](https://github.com/classzz/classzz/blob/master/docs/code_contribution_guidelines.md) before starting.


## License

classzz is licensed under the [copyfree](http://copyfree.org) ISC License.
