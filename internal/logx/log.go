package logx

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func Open(root string) (*log.Logger, func(), error) {
	p, e := layout.Safe(root, "logs")
	if e != nil {
		return nil, nil, e
	}
	if e = os.MkdirAll(p, 0700); e != nil {
		return nil, nil, e
	}
	files, _ := filepath.Glob(filepath.Join(p, "pftw-*.log"))
	sort.Strings(files)
	for len(files) > 9 {
		_ = os.Remove(files[0])
		files = files[1:]
	}
	f, e := os.OpenFile(filepath.Join(p, "pftw-"+time.Now().Format("20060102")+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return nil, nil, e
	}
	return log.New(f, "", log.LstdFlags), func() { _ = f.Close() }, nil
}
