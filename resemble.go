package resemble

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
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
		return errors.New("please provide at least one asset directory")
	}

	o, err := os.Create(r.OutputFile)
	if err != nil {
		return err
	}

	fmt.Fprintf(o, "// Code generated by go-resemble. DO NOT EDIT.\n\n")
	fmt.Fprintf(o, "//lint:file-ignore U1000 This file is automatically generated; ignore unused code checks\n\n")

	fmt.Fprintf(o, "package %s\n\n", r.PackageName)

	if r.Debug {
		return r.dynamicAssets(o)
	} else {
		return r.staticAssets(o)
	}
}

const staticTemplate string = `type embeddedStaticAssets struct {
	prefix string
}

func (e embeddedStaticAssets) Open(name string) (fs.File, error) {
	n := name
	if e.prefix != "" {
		n = e.prefix + "/" + name
	}
	buf, err := getAsset(n)
	if err == nil {
		return embeddedStaticBuffer{
			name: name,
			size: len(buf),
			buf:  bytes.NewReader(buf),
		}, nil
	}

	return nil, fs.ErrNotExist
}
func (e embeddedStaticAssets) Stat() (fs.FileInfo, error) {
	return e, nil
}
func (e embeddedStaticAssets) Read(dest []byte) (int, error) {
	return 0, fmt.Errorf("this is a directory")
}
func (e embeddedStaticAssets) Close() error {
	return nil
}
func (e embeddedStaticAssets) ReadDir(n int) ([]fs.DirEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e embeddedStaticAssets) Name() string       { return e.prefix }
func (e embeddedStaticAssets) Size() int64        { return 0 }
func (e embeddedStaticAssets) Mode() fs.FileMode  { return fs.ModeDir | 0555 }
func (e embeddedStaticAssets) ModTime() time.Time { return time.Unix(embedTime, 0) }
func (e embeddedStaticAssets) IsDir() bool        { return true }
func (e embeddedStaticAssets) Sys() any           { return nil }

type embeddedStaticBuffer struct {
	name string
	size int
	buf  io.Reader
}

func (f embeddedStaticBuffer) Stat() (fs.FileInfo, error) {
	return f, nil
}
func (f embeddedStaticBuffer) Read(dest []byte) (int, error) {
	return f.buf.Read(dest)
}
func (f embeddedStaticBuffer) Close() error {
	return nil
}

func (f embeddedStaticBuffer) Name() string       { return f.name }
func (f embeddedStaticBuffer) Size() int64        { return int64(f.size) }
func (f embeddedStaticBuffer) Mode() fs.FileMode  { return 0444 }
func (f embeddedStaticBuffer) ModTime() time.Time { return time.Unix(embedTime, 0) }
func (f embeddedStaticBuffer) IsDir() bool        { return false }
func (f embeddedStaticBuffer) Sys() any           { return nil }
`

func (r Resemble) staticAssets(o io.WriteCloser) error {
	assets := newCollection()

	for _, aPath := range r.AssetPaths {
		err := assets.AddPath(aPath)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(o, "import \"bytes\"\n")
	fmt.Fprintf(o, "import \"fmt\"\n")
	fmt.Fprintf(o, "import \"io\"\n")
	fmt.Fprintf(o, "import \"io/fs\"\n")
	fmt.Fprintf(o, "import \"time\"\n")
	fmt.Fprintf(o, "\n")

	fmt.Fprintf(o, "const assetsEmbedded bool = true\n")
	fmt.Fprintf(o, "const embedTime int64 = 0x%08x\n", time.Now().Unix())
	fmt.Fprintf(o, "\n")

	fmt.Fprintf(o, "%s\n", staticTemplate)
	fmt.Fprintf(o, "var embeddedAssets embeddedStaticAssets = embeddedStaticAssets {\n")
	fmt.Fprintf(o, "\tprefix: \"\",\n")
	fmt.Fprintf(o, "}\n\n")

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

	dotPath := -1

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

		if p == "." {
			dotPath = i
		}
	}

	fmt.Fprintf(o, "import \"fmt\"\n")
	fmt.Fprintf(o, "import \"io/fs\"\n")
	fmt.Fprintf(o, "import \"io/ioutil\"\n")
	fmt.Fprintf(o, "import \"os\"\n")
	fmt.Fprintf(o, "\n")

	fmt.Fprintf(o, "const assetsEmbedded bool = false\n\n")

	fmt.Fprintf(o, "%s\n", "type embeddedDynamicAssets struct {}\n")
	fmt.Fprintf(o, "%s\n", "func (d embeddedDynamicAssets) Open(name string) (fs.File, error) {\n")
	elseif := "if"
	for i, p := range r.AssetPaths {
		if i == dotPath {
			continue
		}
		abs := absPaths[i]
		fmt.Fprintf(o, "\t%s len(name) >= %d && name[:%d] == \"%s\" {\n", elseif, len(p), len(p), goString(p))
		fmt.Fprintf(o, "\t\treturn os.Open(\"%s\" + name[%d:])\n", goString(abs), len(p))
		elseif = "} else if"
	}
	if dotPath >= 0 {
		fmt.Fprintf(o, "\t%s len(name) > 0 {\n", elseif)
		fmt.Fprintf(o, "\t\treturn os.Open(\"%s\" + name)\n", goString(absPaths[dotPath]))
	}
	fmt.Fprintf(o, "\t} else {\n\t\treturn nil, fs.ErrNotExist\n\t}\n")
	fmt.Fprintf(o, "}\n")
	fmt.Fprintf(o, "\n")

	fmt.Fprintf(o, "var embeddedAssets embeddedDynamicAssets = embeddedDynamicAssets {\n")
	fmt.Fprintf(o, "\t// todo\n")
	fmt.Fprintf(o, "}\n\n")

	fmt.Fprintf(o, "func getAsset(name string) ([]byte, error) {\n")
	fmt.Fprintf(o, "\tvar rvp string\n")

	elseif = "if"
	for i, p := range r.AssetPaths {
		if i == dotPath {
			continue
		}
		abs := absPaths[i]
		fmt.Fprintf(o, "\t%s len(name) >= %d && name[:%d] == \"", elseif, len(p), len(p))
		writeGoString(o, p)
		fmt.Fprintf(o, "\" {\n\t\trvp = \"")
		writeGoString(o, abs)
		fmt.Fprintf(o, "\" + name[%d:]\n", len(p))
		elseif = "} else if"
	}

	// Special case to handle "."
	if dotPath >= 0 {
		fmt.Fprintf(o, "\t%s len(name) > 0 {\n\t\trvp = \"", elseif)
		writeGoString(o, absPaths[dotPath])
		fmt.Fprintf(o, "\" + name\n")
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
func goString(str string) string {
	rv := ""
	for _, b := range str {
		if b < 256 && bytem[int(b)] != nil {
			rv += string(bytem[int(b)])
		} else if b < 32 || b == '\\' || b == '"' {
			rv += fmt.Sprintf("\\x%02x", b)
		} else {
			rv += string(b)
		}
	}
	return rv
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
