package main

import (
	"flag"
	"log"

	"github.com/thijzert/go-resemble"
)

func main() {
	var emb resemble.Resemble
	flag.StringVar(&emb.OutputFile, "o", "assets.go", "Output file name")
	flag.StringVar(&emb.PackageName, "p", "", "Output package name")
	flag.BoolVar(&emb.Debug, "debug", false, "Debug build: generate the getAsset() function stub, but don't actually embed files.")
	flag.Parse()

	emb.AssetPaths = flag.Args()

	if err := emb.Run(); err != nil {
		log.Fatal(err)
	}
}
