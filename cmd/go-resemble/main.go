package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

func main() {
	var outputFile string
	var packageName string
	var debugBuild bool
	flag.StringVar(&outputFile, "o", "assets.go", "Output file name")
	flag.StringVar(&packageName, "p", "", "Output package name")
	flag.BoolVar(&debugBuild, "debug", false, "Debug build: generate the getAsset() function stub, but don't actually embed files.")
	flag.Parse()

	if packageName == "" {
		log.Fatal("TODO: figure out a package name. Until I do that, please supply it with the '-p' parameter")
	}

	assetPaths := flag.Args()
	if len(assetPaths) == 0 {
		log.Fatal("Please provide at least one asset directory")
	}

	if debugBuild {
		dynamicAssets(outputFile, packageName, assetPaths)
	} else {
		staticAssets(outputFile, packageName, assetPaths)
	}
}

func staticAssets(outputFile string, packageName string, assetPaths []string) {
	assets := New()

	for _, aPath := range assetPaths {
		err := assets.AddPath(aPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	o, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(o, "package %s\n\nimport \"fmt\"\n\n", packageName)

	fmt.Fprintf(o, "const assetsEmbedded bool = true\n\n")

	for _, ass := range assets.Assets {
		fmt.Fprintf(o, "var %s string = \"", ass.Varname)
		writeGoBytes(o, ass.Contents)
		fmt.Fprintf(o, "\"\n")
	}
	fmt.Fprintf(o, "\n")

	fmt.Fprintf(o, "func getAsset(name string) ([]byte, error) {\n")
	if len(assets.Assets) > 0 {
		elseif := "if"
		for _, ass := range assets.Assets {
			fmt.Fprintf(o, "\t%s name == \"", elseif)
			writeGoString(o, ass.Path)
			fmt.Fprintf(o, "\" {\n\t\treturn []byte(%s), nil\n", ass.Varname)
			elseif = "} else if"
		}
		fmt.Fprintf(o, "\t} else {\n\t\treturn nil, fmt.Errorf(\"asset not found\")\n\t}\n")
	} else {
		fmt.Fprintf(o, "\treturn nil, fmt.Errorf(\"asset not found\")\n")
	}
	fmt.Fprintf(o, "}\n")
	o.Close()
}

func dynamicAssets(outputFile string, packageName string, assetPaths []string) {
	absPaths := make([]string, len(assetPaths))
	var err error

	for i, p := range assetPaths {
		absPaths[i], err = filepath.Abs(p)
		if err != nil {
			log.Fatal(err)
		}
		fi, err := os.Stat(p)
		if err != nil {
			log.Fatal(err)
		}
		if fi.IsDir() {
			absPaths[i] += "/"
			if p[len(p)-1:] != "/" {
				assetPaths[i] += "/"
			}
		}
	}

	o, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(o, "package %s\n\nimport \"fmt\"\nimport \"io/ioutil\"\n\n", packageName)

	fmt.Fprintf(o, "const assetsEmbedded bool = false\n\n")

	fmt.Fprintf(o, "func getAsset(name string) ([]byte, error) {\n")
	fmt.Fprintf(o, "\tvar rvp string\n")

	elseif := "if"
	for i, p := range assetPaths {
		abs := absPaths[i]
		fmt.Fprintf(o, "\t%s len(name) >= %d && name[:%d] == \"", elseif, len(p), len(p))
		writeGoString(o, p)
		fmt.Fprintf(o, "\" {\n\t\trvp = \"")
		writeGoString(o, abs)
		fmt.Fprintf(o, "\" + name[%d:]\n", len(p))
		elseif = "} else if"
	}
	fmt.Fprintf(o, "\t} else {\n\t\treturn nil, fmt.Errorf(\"asset not found\")\n\t}\n\n")

	fmt.Fprintf(o, "\treturn ioutil.ReadFile(rvp)\n")
	fmt.Fprintf(o, "}\n")
	o.Close()
}

var bytem [][]byte

func writeGoBytes(f io.Writer, str []byte) error {
	for _, b := range str {
		if bytem[int(b)] != nil {
			f.Write(bytem[int(b)])
		} else if b < 32 || b >= 128 || b == '\\' || b == '"' {
			fmt.Fprintf(f, "\\x%02x", b)
		} else {
			f.Write([]byte{b})
		}
	}
	return nil
}
func writeGoString(f io.Writer, str string) error {
	for _, b := range str {
		if b < 256 && bytem[int(b)] != nil {
			f.Write(bytem[int(b)])
		} else if b < 32 || b == '\\' || b == '"' {
			fmt.Fprintf(f, "\\x%02x", b)
		} else {
			fmt.Fprintf(f, "%c", b)
		}
	}
	return nil
}
func init() {
	bytem = make([][]byte, 256)
	bytem[int('\n')] = []byte("\\n")
	bytem[int('\r')] = []byte("\\r")
	bytem[int('\t')] = []byte("\\t")
	bytem[int('\\')] = []byte("\\\\")
	bytem[int('"')] = []byte("\\\"")
}

type Ass struct {
	Path     string
	Basename string
	DirSplit []string
	Varname  string
	Contents []byte
}

type AssCollection struct {
	Assets []Ass
}

func New() *AssCollection {
	rv := &AssCollection{
		Assets: make([]Ass, 0),
	}
	return rv
}

func (ac *AssCollection) Add(a Ass) error {
	a.Basename = path.Base(a.Path)
	a.DirSplit = filepath.SplitList(path.Dir(a.Path))
	a.Varname = varname(a.Path)

	for _, b := range ac.Assets {
		if b.Path == a.Path {
			return fmt.Errorf("Duplicate file name '%s'", a.Path)
		} else if b.Varname == a.Varname {
			return fmt.Errorf("Duplicate variable name '%s' → %s", a.Path, a.Varname)
		}
	}

	ac.Assets = append(ac.Assets, a)

	return nil
}

func (ac *AssCollection) AddPath(aPath string) error {
	fi, err := os.Stat(aPath)
	if err != nil {
		return err
	}
	f, err := os.Open(aPath)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		dn, err := f.Readdirnames(-1)
		if err != nil {
			return err
		}
		for _, chd := range dn {
			err := ac.AddPath(path.Join(aPath, chd))
			if err != nil {
				return err
			}
		}
	} else {
		cnt, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		ass := Ass{
			Path:     aPath,
			Contents: cnt,
		}
		err = ac.Add(ass)
		if err != nil {
			return err
		}
	}

	return nil
}

func varname(pt string) string {
	var rv bytes.Buffer
	rv.WriteString("__")
	for _, c := range pt {
		if (c >= '0' && c <= '9') ||
			(c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') {
			fmt.Fprintf(&rv, "%c", c)
		} else {
			fmt.Fprintf(&rv, "_%2X", int(c))
		}
	}
	return rv.String()
}
