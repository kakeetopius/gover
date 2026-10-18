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
gover update --force # update even if the latest version is already installed.

# switch to a specific go version. If the version is not found offline, it is downloaded (if it is available).
gover switch 1.26.9
```
