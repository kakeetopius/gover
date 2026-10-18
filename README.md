# gover

Manage go versions installed on the system.

## Installation

```bash
go install github.com/kakeetopius/gover@latest

#OR
git clone https://github.com/kakeetopius/gover.git && cd gover
go install .
```

## Usage

```bash
gover update # update to the latest stable go version.
gover update --unstable # allow updating to an unstable version if it is the latest.

# switch to a specific go version. If the version is not found offline, it is downloaded (if it is available online).
gover switch go1.26.9
gover switch 1.24.0

gover list # list installed versions
gover list --all # list all available versions even if not installed offline.

gover purge go1.19.4 # remove a particular go version. (Only removes versions that were installed by gover)
gover purge --all # remove all go versions that were installed by gover.
```
