package platform

import "testing"

func TestElevationProbe(t *testing.T) {
	v, e := (Native{}).IsElevated()
	if e != nil {
		t.Fatal(e)
	}
	if v {
		t.Skip("test host is elevated")
	}
	free, e := (Native{}).FreeBytes(t.TempDir())
	if e != nil || free == 0 {
		t.Fatal(free, e)
	}
}

func TestGitBashPTYNames(t *testing.T) {
	for _, name := range []string{`\msys-1888ae32-pty0-from-master`, `\cygwin-a123-pty12-to-master`} {
		if !isMsysPTY(name) {
			t.Fatal("missed terminal", name)
		}
	}
	for _, name := range []string{`\other-pipe`, `\msys-abcd-not-a-pty`, `C:\file.txt`} {
		if isMsysPTY(name) {
			t.Fatal("misclassified redirection", name)
		}
	}
}
