# GoFind

A simple utility to find files and directories written in Go.

## Installation

```bash
go install github.com/benjkr/gofind@latest
```

## Usage

```
gofind [ARGUMENTS] FOLDER
```

### Arguments

```
-H                     human readable sizes [default: false]
--inverted, -i         inverted sort [default: false]
--depth DEPTH          max depth to go (0 = infinite) [default: 0]
--all, -a              do not ignore entries starting with .
--tree, -t             Prints Tree of all indexed files
--top TOP              N Top files [default: 0]
--dirs, -d             include directories [default: false]
--verbose, -v          verbose mode [default: false]
```

### Examples

Find all files in the current directory:

```bash
gofind .
```

Find all files in the current directory, sorted by size (Smallest first):

```bash
gofind . --inverted
```

Tree of all files in the current directory:

```bash
gofind . --tree
```

Find all files in home directory, including hidden files and directories, and prints sizes in human readable format:

```bash
gofind ~ -d -a -H

# Prints the top 10 files
gofind ~ -d -a -H --top 10
```
