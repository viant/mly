## Mly tools

### Installation

```bash
git clone https://github.com/viant/mly.git
cd mly/tools/mlytool
go build
cp mlytool /usr/local/bin
```

### Usage

```bash
mlytool -h
```

##### Exporting layers

```bash
mlytool -m=discover -s=~/model/ -d=layers.json -o=layers
```
