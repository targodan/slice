# slice

Tiny tool to output portions of binary files with different formats.

## How to install

```bash
go get github.com/targodan/slice
```

## Usage

```
NAME:
   slice - outputs contents of binary files

USAGE:
   slice [options] FILE

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --offset value, -o value                          offset of output in bytes (default: "0")
   --size value, --length value, -s value, -l value  size of output in bytes (default: "-1")
   --format value, -f value                          output format, available: raw, hex, dump, gobytes, gostring, cstring, base64, md5, sha256 (default: "raw")
   --help, -h                                        show help (default: false)
```
