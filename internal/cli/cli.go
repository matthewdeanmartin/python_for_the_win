package cli

import (
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/install"
	system "github.com/matthewdeanmartin/python_for_the_win/internal/integration"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/logx"
	"github.com/matthewdeanmartin/python_for_the_win/internal/overlay"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/preflight"
	"github.com/matthewdeanmartin/python_for_the_win/internal/proc"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/selfupdate"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var Version = "dev"
var Commit = "unknown"
var Date = "unknown"

type options struct {
	step                                                                                        func(string)
	progress                                                                                    func(int64, int64)
	info                                                                                        platform.Info
	store                                                                                       envpath.Store
	self                                                                                        string
	allowAdmin, portable, setDefaultFlag                                                        bool
	dryRun                                                                                      bool
	fresh, showUnavailable                                                                      bool
	root, source, proxy, arch, path                                                             string
	yes, quiet, verbose, json, insecure, refresh, pre, threaded, noPip, all, available, allArch bool
}

func New() *cobra.Command {
	return NewWith(platform.Native{}, envpath.UserStore(), "")
}

func NewWith(info platform.Info, store envpath.Store, self string) *cobra.Command {
	o := &options{info: info, store: store, self: self}
	root := &cobra.Command{Use: "pftw", Short: "Install official Python without a system installer", SilenceUsage: true, SilenceErrors: true, RunE: func(c *cobra.Command, _ []string) error {
		if o.dryRun {
			return errs.New(2, "Use an explicit command with --dry-run, such as install")
		}
		if runtime.GOOS == "windows" && o.interactive() {
			return o.wizard(c)
		}
		_ = c.Help()
		return errs.New(2, "Use an explicit command when input or output is redirected.")
	}}
	f := root.PersistentFlags()
	f.BoolVar(&o.dryRun, "dry-run", false, "Report changes without writing files or settings")
	f.BoolVar(&o.dryRun, "dry-mode", false, "Alias for --dry-run")
	f.BoolVar(&o.allowAdmin, "allow-admin", false, "Explicitly allow an elevated process")
	f.StringVar(&o.root, "root", "", "Installation root (default: user AppData/pftw)")
	f.StringVar(&o.source, "source", "", "Index URL, file, or directory")
	f.StringVar(&o.proxy, "proxy", "", "HTTPS proxy URL")
	f.BoolVar(&o.insecure, "allow-insecure-source", false, "Allow HTTP sources (sha256 still required)")
	f.BoolVar(&o.refresh, "refresh", false, "Bypass index cache")
	f.BoolVar(&o.yes, "yes", false, "Accept non-decision prompts")
	f.BoolVar(&o.quiet, "quiet", false, "Hide progress")
	f.BoolVar(&o.verbose, "verbose", false, "Show extra details")
	f.BoolVar(&o.json, "json", false, "Emit JSON output")
	root.SetFlagErrorFunc(func(_ *cobra.Command, e error) error { return errs.Wrap(2, "invalid arguments", e) })
	add := func(use, short string, args cobra.PositionalArgs, run func(*cobra.Command, []string) error) *cobra.Command {
		c := &cobra.Command{Use: use, Short: short, Args: func(c *cobra.Command, a []string) error {
			if e := args(c, a); e != nil {
				return errs.Wrap(2, "invalid arguments", e)
			}
			return nil
		}, RunE: run}
		root.AddCommand(c)
		return c
	}
	add("gui", "Open the graphical installation wizard", cobra.NoArgs, func(c *cobra.Command, _ []string) error {
		if runtime.GOOS != "windows" {
			return errs.New(8, "The graphical wizard is currently supported only on Windows")
		}
		if o.dryRun {
			return errs.New(2, "Use install --dry-run to preview installation without opening a wizard")
		}
		if o.json || o.yes || o.quiet {
			return errs.New(2, "gui is interactive; omit --json, --yes and --quiet")
		}
		return o.runWizard(c, true)
	})
	ins := add("install [SPEC]", "Install Python (latest final by default)", cobra.MaximumNArgs(1), func(c *cobra.Command, a []string) error { return o.install(c, a) })
	ins.Flags().StringVar(&o.path, "path", "auto", "PATH policy: auto, first, or none")
	ins.Flags().BoolVar(&o.portable, "portable", false, "Contain files in --root and do not manage PATH")
	ins.Flags().BoolVar(&o.setDefaultFlag, "set-default", false, "Make this runtime the default")
	ins.Flags().StringVar(&o.arch, "arch", platform.NativeArch(), "amd64, arm64, or x86")
	ins.Flags().BoolVar(&o.threaded, "freethreaded", false, "Select free-threaded Python")
	ins.Flags().BoolVar(&o.pre, "prerelease", false, "Allow development releases")
	ins.Flags().BoolVar(&o.noPip, "no-pip", false, "Skip pip installation and verification")
	ins.Flags().BoolVar(&o.fresh, "fresh", false, "Omit existing packages when upgrading")
	up := add("upgrade [SPEC]", "Upgrade within installed minor versions", cobra.MaximumNArgs(1), o.upgrade)
	up.Flags().BoolVar(&o.all, "all", false, "Upgrade every managed runtime")
	up.Flags().BoolVar(&o.fresh, "fresh", false, "Omit existing packages and Scripts")
	ls := add("list", "List installed or available Python releases", cobra.NoArgs, func(c *cobra.Command, a []string) error { return o.list(c) })
	ls.Flags().BoolVar(&o.showUnavailable, "show-unavailable", false, "Include unsupported entries with exact reasons")
	ls.Flags().BoolVar(&o.available, "available", false, "Show feed releases")
	ls.Flags().BoolVar(&o.pre, "prerelease", false, "Show all development releases")
	ls.Flags().BoolVar(&o.allArch, "all-arch", false, "Show all architectures")
	un := add("uninstall [SPEC]", "Remove a pftw-owned Python", cobra.MaximumNArgs(1), func(c *cobra.Command, a []string) error {
		if e := o.uninstall(c, a); e != nil {
			return e
		}
		if o.all && !o.dryRun {
			return o.removeManager(c)
		}
		return nil
	})
	un.Flags().BoolVar(&o.all, "all", false, "Remove all managed runtimes")
	add("config ACTION [KEY] [VALUE]", "Read or change source and proxy settings", cobra.RangeArgs(1, 3), func(c *cobra.Command, a []string) error { return o.config(c, a) })
	add("default SPEC", "Choose the default installed Python", cobra.ExactArgs(1), o.setDefault)
	var fix bool
	doc := add("doctor", "Explain installation and command resolution", cobra.NoArgs, func(c *cobra.Command, a []string) error {
		if fix {
			return o.doctorFix(c)
		}
		return o.doctor(c)
	})
	doc.Flags().BoolVar(&fix, "fix", false, "Preview and repair missing pftw-owned commands and PATH entries")
	add("version", "Show build information", cobra.NoArgs, func(c *cobra.Command, a []string) error {
		return o.output(c, map[string]string{"version": Version, "commit": Commit, "date": Date, "go": runtime.Version(), "arch": runtime.GOARCH, "native_arch": platform.NativeArch()}, fmt.Sprintf("pftw %s (%s), %s, %s (native %s)", Version, Commit, runtime.Version(), runtime.GOARCH, platform.NativeArch()))
	})
	root.AddCommand(&cobra.Command{Use: "docs", Hidden: true, Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error { return CLIReference(c.OutOrStdout(), root) }})
	var force bool
	update := add("self-update", "Update pftw from a verified GitHub release", cobra.NoArgs, func(c *cobra.Command, _ []string) error { return o.selfUpdate(c, force) })
	update.Flags().BoolVar(&force, "force", false, "Allow replacing a development build")
	update.Flags().BoolVar(&o.pre, "prerelease", false, "Include prerelease pftw releases")
	root.AddCommand(o.runCommand(), o.bundleCommand(), o.mirrorCommand())
	return root
}
func (o *options) output(c *cobra.Command, value any, message string) error {
	if o.json {
		return json.NewEncoder(c.OutOrStdout()).Encode(value)
	}
	if !o.quiet {
		fmt.Fprintln(c.OutOrStdout(), message)
	}
	return nil
}
func (o *options) load() (*state.State, error) {
	r, e := o.rootPath()
	if e != nil {
		return nil, e
	}
	o.root = r
	return state.Load(r)
}
func (o *options) client(s *state.State) (*download.Client, string, error) {
	proxy := o.proxy
	if proxy == "" && os.Getenv("HTTPS_PROXY") == "" && os.Getenv("https_proxy") == "" {
		proxy = s.Setting("proxy")
	}
	d, e := download.New(proxy, o.insecure)
	if e != nil {
		return nil, "", errs.Wrap(2, "proxy configuration", e)
	}
	exe := o.self
	if exe == "" {
		exe, _ = os.Executable()
	}
	sourceFlag := o.source
	if sourceFlag == "" && os.Getenv("PFTW_SOURCE") == "" {
		info, err := overlay.Inspect(exe)
		if err != nil {
			return nil, "", errs.Wrap(6, "cannot inspect offline bundle", err)
		}
		if info != nil {
			sourceFlag = overlay.Source(exe, "index.json")
		}
	}
	source, _ := feed.Source(sourceFlag, os.Getenv("PFTW_SOURCE"), filepath.Dir(exe), s.Setting("source"))
	source, e = feed.Normalize(source)
	return d, source, e
}
func (o *options) entries(c *cobra.Command, s *state.State) ([]feed.Entry, *download.Client, string, error) {
	d, source, e := o.client(s)
	if e != nil {
		return nil, nil, "", e
	}
	cache, e := layout.Safe(o.root, "cache")
	if e != nil {
		return nil, nil, "", e
	}
	if o.dryRun {
		cache = ""
	}
	it, e := feed.Load(d, source, cache, o.refresh)
	if e != nil {
		return nil, nil, "", e
	}
	v, e := it.All(c.Context())
	return v, d, source, e
}
func (o *options) mutate(c *cobra.Command, fn func(*state.State, *install.Manager) error) error {
	if o.dryRun {
		return o.plan(c, c.Flags().Args())
	}
	if runtime.GOOS != "windows" {
		return errs.New(8, "Runtime installation is supported only on Windows")
	}
	self := o.self
	if self == "" {
		self, _ = os.Executable()
	}
	selfupdate.Cleanup(self)
	r, e := o.rootPath()
	if e != nil {
		return e
	}
	o.root = r
	if e = o.checks(c, preflight.Run(o.info, r, "", state.New(), o.allowAdmin)); e != nil {
		return e
	}
	unlock, e := state.Lock(r)
	if e != nil {
		return e
	}
	defer unlock()
	logger, closeLog, e := logx.Open(r)
	if e != nil {
		return e
	}
	defer closeLog()
	logger.Printf("command %s", c.Name())
	if e = install.Recover(r); e != nil {
		return e
	}
	if e = system.Recover(r, o.store); e != nil {
		return e
	}
	if e = install.Cleanup(r); e != nil {
		logger.Printf("cleanup deferred: %v", e)
	}
	s, e := state.Load(r)
	if e != nil {
		return e
	}
	m := &install.Manager{FreeBytes: o.info.FreeBytes, Root: r, Runner: proc.Real{}, Log: logger, Step: func(s string) {
		if o.step != nil {
			o.step(s)
		}
		if !o.quiet {
			fmt.Fprintln(c.ErrOrStderr(), s)
		}
	}}
	e = fn(s, m)
	if e != nil {
		logger.Printf("failed: %v", e)
	}
	return e
}
func (o *options) install(c *cobra.Command, a []string) error {
	if o.path != "none" && o.path != "auto" && o.path != "first" {
		return errs.New(2, "--path must be auto, first, or none")
	}
	if o.portable {
		if !c.Flags().Changed("root") {
			return errs.New(2, "--portable requires --root")
		}
		if o.path == "first" {
			return errs.New(2, "--portable cannot use --path first")
		}
		o.path = "none"
	}
	spec := "latest"
	if len(a) > 0 {
		spec = a[0]
	}
	opts := resolve.Options{Spec: spec, Arch: o.arch, Threaded: o.threaded, Prerelease: o.pre}
	if _, _, e := resolve.Normalize(opts); e != nil {
		return errs.Wrap(2, "invalid version selector", e)
	}
	return o.mutate(c, func(s *state.State, m *install.Manager) error {
		entries, d, source, e := o.entries(c, s)
		if e != nil {
			return e
		}
		v, e := resolve.Pick(entries, opts)
		if e != nil {
			return errs.Wrap(3, "cannot select Python", e)
		}
		if e = o.checks(c, preflight.Run(o.info, o.root, v.Key(), s, o.allowAdmin)); e != nil {
			return e
		}
		if o.portable {
			if s.Path().Mode == "first" {
				return errs.New(2, "This root already manages PATH; choose a new portable root")
			}
			s.SetPortable(true)
			if e = s.Save(o.root); e != nil {
				return e
			}
		}
		mode, e := o.choosePath(c, s)
		if e != nil {
			return e
		}
		pv, _ := pyver.Parse(v.Version)
		if pv.Pre() {
			fmt.Fprintf(c.ErrOrStderr(), "Development release: Python %s (explicit version selection).\n", v.Version)
		}
		if o.source != "" && source != feed.DefaultSource {
			fmt.Fprintln(c.ErrOrStderr(), "Whoever controls the source controls what gets installed.")
		}
		if !o.quiet && platform.IsTerminal(os.Stderr) {
			last := -1
			d.Progress = func(done, total int64) {
				if total > 0 {
					percent := int(done * 100 / total)
					if percent != last {
						fmt.Fprintf(c.ErrOrStderr(), "\rDownloading: %d%%", percent)
						last = percent
						if done == total {
							fmt.Fprintln(c.ErrOrStderr())
						}
					}
				}
			}
		}
		if o.progress != nil {
			d.Progress = o.progress
		}
		m.Fresh = o.fresh
		m.Download = d
		message, e := m.Install(c.Context(), v, source, o.noPip)
		if e != nil {
			return e
		}
		s, e = state.Load(o.root)
		if e != nil {
			return e
		}
		if o.setDefaultFlag {
			s.Default = v.Key()
		}
		if e = o.sync(c, s, mode == "first" || s.Path().Mode == "first"); e != nil {
			return e
		}
		if o.pathResolvesDefault(s) {
			message = strings.Replace(message, "PATH was not changed.\n", "", 1) + "\nOpen a new terminal, then run: python --version"
		}
		return o.output(c, map[string]string{"message": message, "root": o.root}, message)
	})
}
func (o *options) list(c *cobra.Command) error {
	s, e := o.load()
	if e != nil {
		return e
	}
	_, source, e := o.client(s)
	if e != nil {
		return e
	}
	if o.available {
		v, _, src, e := o.entries(c, s)
		if e != nil {
			return e
		}
		v = resolve.Available(v, platform.NativeArch(), o.allArch, o.pre, o.showUnavailable)
		if o.json {
			return o.output(c, map[string]any{"source": src, "versions": availabilityJSON(v)}, "")
		}
		if src != feed.DefaultSource {
			fmt.Fprintln(c.OutOrStdout(), "Source:", src)
		}
		for _, x := range v {
			ver, _ := pyver.Parse(x.Version)
			note := ""
			if ver.Pre() {
				note = " pre"
			}
			if !x.Installable() {
				note += " unavailable: " + strings.Join(x.UnavailableReasons(), "; ")
			}
			fmt.Fprintf(c.OutOrStdout(), "%-18s %-15s%s\n", x.Key(), x.Version, note)
		}
		return nil
	}
	if o.json {
		return o.output(c, map[string]any{"source": source, "default": s.Default, "installs": s.Installs}, "")
	}
	if source != feed.DefaultSource {
		fmt.Fprintln(c.OutOrStdout(), "Source:", source)
	}
	keys := []string{}
	for k := range s.Installs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v, e := s.Get(k)
		if e != nil {
			return e
		}
		mark := " "
		if k == s.Default {
			mark = "*"
		}
		fmt.Fprintf(c.OutOrStdout(), "%s %-18s %s\n", mark, k, v.Version)
	}
	if len(keys) == 0 {
		fmt.Fprintln(c.OutOrStdout(), "No Python versions installed by pftw.")
	}
	return nil
}
func (o *options) uninstall(c *cobra.Command, a []string) error {
	if (len(a) == 0 && !o.all) || (len(a) > 0 && o.all) {
		return errs.New(2, "choose SPEC or --all")
	}
	return o.mutate(c, func(s *state.State, m *install.Manager) error {
		keys := []string{}
		if o.all {
			for k := range s.Installs {
				keys = append(keys, k)
			}
		} else {
			spec, w, e := resolve.Normalize(resolve.Options{Spec: a[0], Arch: platform.NativeArch()})
			if e != nil {
				return errs.Wrap(2, "invalid selector", e)
			}
			for k := range s.Installs {
				v, e := s.Get(k)
				if e != nil {
					return e
				}
				if k == spec || (w.Parts == 3 && strings.TrimSuffix(spec, "-"+resolve.Suffix(v.Arch)) == v.Version) {
					keys = append(keys, k)
				}
			}
			if len(keys) != 1 {
				return errs.New(2, "select exactly one installed minor/variant/architecture; use list")
			}
		}
		replacement, e := o.chooseReplacement(c, s, keys)
		if e != nil {
			return e
		}
		sort.Strings(keys)
		for _, k := range keys {
			if e := m.Uninstall(k); e != nil {
				return e
			}
		}
		s, e = state.Load(o.root)
		if e != nil {
			return e
		}
		if replacement != "" {
			s.Default = replacement
		}
		if e = o.sync(c, s, len(s.Installs) > 0 && s.Path().Mode == "first"); e != nil {
			return e
		}
		return o.output(c, map[string]any{"removed": keys}, "Removed managed runtimes and their command/PATH entries.")
	})
}
func (o *options) config(c *cobra.Command, a []string) error {
	action := a[0]
	if action != "list" && action != "get" && action != "set" && action != "unset" {
		return errs.New(2, "config action must be list, get, set, or unset")
	}
	want := 2
	if action == "list" {
		want = 1
	}
	if action == "set" {
		want = 3
	}
	if len(a) != want {
		return errs.New(2, "wrong number of config arguments")
	}
	if len(a) > 1 && a[1] != "source" && a[1] != "proxy" {
		return errs.New(2, "unknown config key %q", a[1])
	}
	s, e := o.load()
	if e != nil {
		return e
	}
	if action == "list" {
		return json.NewEncoder(c.OutOrStdout()).Encode(s.Config)
	}
	if action == "get" {
		return o.output(c, s.Setting(a[1]), s.Setting(a[1]))
	}
	return o.mutate(c, func(s *state.State, m *install.Manager) error {
		if action == "unset" {
			delete(s.Config, a[1])
		} else {
			value := a[2]
			if a[1] == "source" {
				value, e = feed.Normalize(value)
				if e != nil {
					return e
				}
				d, _, e := o.client(s)
				if e != nil {
					return e
				}
				it, e := feed.Load(d, value, "", true)
				if e != nil {
					return e
				}
				if _, e = it.Next(c.Context()); e != nil {
					return e
				}
				if value != feed.DefaultSource {
					fmt.Fprintln(c.ErrOrStderr(), "Whoever controls the source controls what gets installed.")
				}
			} else {
				if _, e = download.New(value, o.insecure); e != nil {
					return errs.Wrap(2, "invalid proxy", e)
				}
			}
			s.Config[a[1]], e = json.Marshal(value)
			if e != nil {
				return e
			}
		}
		return s.Save(o.root)
	})
}

func availabilityJSON(entries []feed.Entry) []map[string]any {
	out := []map[string]any{}
	for _, v := range entries {
		b, _ := json.Marshal(v)
		row := map[string]any{}
		_ = json.Unmarshal(b, &row)
		row["installable"] = v.Installable()
		row["unavailable_reasons"] = v.UnavailableReasons()
		out = append(out, row)
	}
	return out
}
