package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/thijzert/go-resemble"
)

func main() {
	outputFile := "assets.go"
	var emb resemble.Resemble
	flag.StringVar(&outputFile, "o", "assets.go", "Output file name")
	flag.StringVar(&emb.PackageName, "p", "", "Output package name")
	flag.BoolVar(&emb.Debug, "debug", false, "Debug build: generate the getAsset() function stub, but don't actually embed files.")
	flag.Parse()

	emb.AssetPaths = flag.Args()

	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := emb.Run(context.Background(), f); err != nil {
		log.Fatal(err)
	}
}
