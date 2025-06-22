<div align="center">
<pre>

  __                                      
_/  |_ __ __  ____   _____ _____    ____  
\   __\  |  \/    \ /     \\__  \  /    \ 
 |  | |  |  /   |  \  Y Y  \/ __ \|   |  \
 |__| |____/|___|  /__|_|  (____  /___|  /
                 \/      \/     \/     \/ 

</pre>

</div>

## Overview

**Tunman** is a CLI tool for managing and automating SSH tunnels. It is designed for developers, DevOps engineers, and anyone who frequently works with port forwarding or remote servers.

This repository includes both the CLI and the Daemon components:

- **Daemon**: Runs in the background to handle all SSH tunnels. It exposes a [gRPC](https://grpc.io/) control plane via a Unix socket and persists tunnel configurations using [bbolt](https://github.com/etcd-io/bbolt).
- **CLI**: A client that interacts with the daemon to create, manage, and monitor tunnels.

Tunnel configurations are persisted, so when the daemon starts, it automatically restores all previously added tunnels, ensuring seamless and continuous connectivity.


## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/Phillezi/tunman/main/scripts/install.sh | bash
```
Check out what the script does [here](https://github.com/Phillezi/tunman/blob/main/scripts/install.sh).

## Usage

Command usage documentation is auto generated using cobra and available [here](./docs/tunman.md).

You can also run:
```bash
tunman --help
```
To see available commands and options.

TODO: continue
