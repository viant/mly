# Mly tools (v1)

Provided in setup and run assistance.

## System Installation

```bash
git clone https://github.com/viant/mly.git
( cd mly/tools/mly
  go build
  cp -v mlytool /usr/local/bin/mly )
```

# Usage

```bash
mly -h
```

## Exporting layers

```bash
mly -m=discover -s=~/model/ -d=layers.json -o=layers
```

