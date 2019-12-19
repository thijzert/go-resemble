package resemble

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// The Resemble wraps all options for EMBedding RESources into a static file
type Resemble struct {
	// The resulting output file (e.g. 'assets.go')
	OutputFile string

	// The package name where the output file lives (e.g. 'fooserver')
	PackageName string

	// For development builds, rather than embedding file contents in source,
	// add a wrapper with the same interface that loads files from disk
	Debug bool

	// Root paths to every asset to be embedded. Directories are traversed recursively.
	AssetPaths []string
}

// Run creates the output file containing all static resources
func (r Resemble) Run() error {
	if r.PackageName == "" {
		return errors.New("TODO: figure out a package name. Until I do that, please supply it with the '-p' parameter")
	}

	if len(r.AssetPaths) == 0 {
		return errors.New("Please provide at least one asset directory")
	}

	o, err := os.Create(r.OutputFile)
	if err != nil {
		return err
	}
	fmt.Fprintf(o, "package %s\n\n", r.PackageName)

	if r.Debug {
		return r.dynamicAssets(o)
	} else {
		return r.staticAssets(o)
	}
}

func (r Resemble) staticAssets(o io.WriteCloser) error {
	assets := newCollection()

	for _, aPath := range r.AssetPaths {
		err := assets.AddPath(aPath)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(o, "import \"fmt\"\n\n")

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
	return o.Close()
}

func (r Resemble) dynamicAssets(o io.WriteCloser) error {
	absPaths := make([]string, len(r.AssetPaths))
	var err error

	for i, p := range r.AssetPaths {
		absPaths[i], err = filepath.Abs(p)
		if err != nil {
			return err
		}
		fi, err := os.Stat(p)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			absPaths[i] += "/"
			if p[len(p)-1:] != "/" {
				r.AssetPaths[i] += "/"
			}
		}
	}

	fmt.Fprintf(o, "import \"fmt\"\nimport \"io/ioutil\"\n\n")

	fmt.Fprintf(o, "const assetsEmbedded bool = false\n\n")

	fmt.Fprintf(o, "func getAsset(name string) ([]byte, error) {\n")
	fmt.Fprintf(o, "\tvar rvp string\n")

	elseif := "if"
	for i, p := range r.AssetPaths {
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
	return o.Close()
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
