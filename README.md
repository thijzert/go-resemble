`go-resemble` is a RESource EMBedder Like Every other.

Installation
------------

    go get -u github.com/thijzert/go-resemble/...

Usage
-----
To embed all files in a directory `assets/`, run:

    go-resemble -o assets.go assets

This writes a file `assets.go` which contains all files in the earlier directory. This file defines a function `getAsset(name string) ([]byte, error)` that can be used to retrieve the original file's contents.

You can also create a debug build like this:

    go-resemble -o assets.go assets

A debug build creates the same `getAsset` function, but keeps reading the files from disk. This is useful during development.
