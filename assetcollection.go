package resemble

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type ass struct {
	Path     string
	Basename string
	DirSplit []string
	Varname  string
	Contents []byte
}

type assCollection struct {
	Assets []ass
}

func newCollection() *assCollection {
	rv := &assCollection{
		Assets: make([]ass, 0),
	}
	return rv
}

func (ac *assCollection) Add(a ass) error {
	a.Basename = path.Base(a.Path)
	a.DirSplit = filepath.SplitList(path.Dir(a.Path))
	a.Varname = varname(a.Path)

	for _, b := range ac.Assets {
		if b.Path == a.Path {
			return fmt.Errorf("duplicate file name '%s'", a.Path)
		} else if b.Varname == a.Varname {
			return fmt.Errorf("duplicate variable name '%s' → %s", a.Path, a.Varname)
		}
	}

	ac.Assets = append(ac.Assets, a)

	return nil
}

func (ac *assCollection) AddPath(aPath string) error {
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

		ass := ass{
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
