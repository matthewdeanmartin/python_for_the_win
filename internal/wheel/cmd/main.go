package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/matthewdeanmartin/python_for_the_win/internal/wheel"
)

func main() {
	s := wheel.Spec{}
	out := flag.String("out", "dist/wheels", "output directory")
	flag.StringVar(&s.Version, "version", "", "PEP 440 version, e.g. 0.3.0")
	flag.StringVar(&s.Platform, "platform", "", "win_amd64, win_arm64 or win32 (or amd64, arm64, 386)")
	flag.StringVar(&s.Manager, "pftw", "", "path to pftw.exe")
	flag.StringVar(&s.Windowed, "pftww", "", "optional path to pftww.exe")
	flag.StringVar(&s.Readme, "readme", "README.md", "project description")
	flag.StringVar(&s.License, "license", "LICENSE", "license file")
	flag.Parse()
	if s.Version == "" || s.Platform == "" || s.Manager == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: wheel -version V -platform TAG -pftw pftw.exe [-pftww pftww.exe] [-out DIR]")
		os.Exit(2)
	}
	if tag, ok := wheel.Platforms[s.Platform]; ok { // Also accept Go architecture names.
		s.Platform = tag
	}
	path, e := wheel.Build(*out, s)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println(path)
}
