`go-resemble` is a RESource EMBedder Like Every other.

Installation
------------

    go get -u github.com/thijzert/go-resemble/...

Command-line usage
------------------
To embed all files in a directory `assets/`, run:

    go-resemble -o assets.go assets

This writes a file `assets.go` which contains all files in the earlier directory. This file defines a function `getAsset(name string) ([]byte, error)` that can be used to retrieve the original file's contents.

You can also create a debug build like this:

    go-resemble -debug -o assets.go assets

A debug build creates the same `getAsset` function, but keeps reading the files from disk. This is useful during development.

As a library
------------
`go-resemble` can be used as a library, for instance, in a build script.

```go
package main

import "context"
import "log"
import "os"
import "os/exec"

func main() {
	ctx := context.Background()
	f, _ := os.Create("assets.go")

	var emb resemble.Resemble
	emb.PackageName = "main"
	emb.Debug = true
	emb.AssetPaths = []string{"assets"}
	if err := emb.Run(ctx, f); err != nil {
		log.Fatalf("error running 'resemble': %v", err)
	}

	c := exec.CommandContext(ctx, "go", "build", "-o", "webserver", "webserver.go", "assets.go")
	if err := c.Run(); err != nil {
		log.Fatalf("compilation error: %v", err)
	}
}
```

License
-------
Copyright (c) 2019-2023 Thijs van Dijk, all rights reserved. Redistribution and use is permitted under the terms of the BSD 3-clause (revised) license. See the file `LICENSE` or [this url](https://tldrlegal.com/license/bsd-3-clause-license-%28revised%29) for details.
