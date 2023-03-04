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

type listEntry struct {
	Basename string
	Size     int64
	IsDir    bool
}

type assCollection struct {
	Assets  []ass
	Listing map[string][]listEntry
}

func newCollection() *assCollection {
	rv := &assCollection{
		Assets:  make([]ass, 0),
		Listing: make(map[string][]listEntry),
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
		dirFis, err := f.Readdir(-1)
		if err != nil {
			return err
		}
		var listing []listEntry
		for _, childFi := range dirFis {
			listing = append(listing, listEntry{
				Basename: childFi.Name(),
				Size:     childFi.Size(),
				IsDir:    childFi.IsDir(),
			})
		}
		if aPath == "." {
			ac.Listing[""] = listing
		} else {
			ac.Listing[aPath] = listing
		}
		for _, childFi := range dirFis {
			err := ac.AddPath(path.Join(aPath, childFi.Name()))
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
