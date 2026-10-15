package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/runner"
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"github.com/spf13/cobra"
)

func (o *options) runCommand() *cobra.Command {
	var python string
	var installMissing bool
	c := &cobra.Command{Use: "run [--python SPEC|PATH] [--install] [SCRIPT | -m MODULE | -c CODE] [ARGS...]", Short: "Run an installed Python; preserve script arguments", DisableFlagParsing: true}
	c.Flags().StringVar(&python, "python", "", "Installed version or interpreter path")
	c.Flags().BoolVar(&installMissing, "install", false, "Explicitly install a missing selected version without PATH changes")
	c.RunE = func(c *cobra.Command, args []string) error {
		c.Flags().AddFlagSet(c.InheritedFlags())
		end := 0
		for end < len(args) {
			arg := args[end]
			if arg == "--" {
				break
			}
			if arg == "-m" || arg == "-c" || !strings.HasPrefix(arg, "-") || arg == "-" {
				break
			}
			name := strings.TrimPrefix(strings.SplitN(arg, "=", 2)[0], "--")
			f := c.Flags().Lookup(name)
			if f == nil {
				return errs.New(2, "unknown runner option %s; pftw flags must precede the script", arg)
			}
			end++
			if f.NoOptDefVal == "" && !strings.Contains(arg, "=") {
				if end == len(args) {
					return errs.New(2, "missing value for %s", arg)
				}
				end++
			}
		}
		if e := c.Flags().Parse(args[:end]); e != nil {
			return errs.Wrap(2, "runner arguments", e)
		}
		if help, _ := c.Flags().GetBool("help"); help {
			return c.Help()
		}
		args = args[end:]
		if len(args) > 0 && args[0] == "--" {
			args = args[1:]
		}
		if len(args) == 1 && (args[0] == "-m" || args[0] == "-c") {
			return errs.New(2, "%s requires a value", args[0])
		}
		s, e := o.load()
		if e != nil {
			return e
		}
		selected, e := runner.Select(o.root, s, python, args)
		var coded *errs.Error
		if errors.As(e, &coded) && coded.Code == 9 && selected.Spec != "" {
			consent := installMissing
			if !consent && o.interactive() {
				fmt.Fprint(c.ErrOrStderr(), e.Error()+". Install it now? [y/N]: ")
				line, _ := bufio.NewReader(c.InOrStdin()).ReadString('\n')
				consent = strings.EqualFold(strings.TrimSpace(line), "y") || strings.EqualFold(strings.TrimSpace(line), "yes")
			}
			if consent {
				spec := selected.Spec
				if selected.Reason == "723" {
					entries, _, _, err := o.entries(c, s)
					if err != nil {
						return err
					}
					best := ""
					var bestVersion pyver.Version
					for _, v := range entries {
						if !v.Installable() || strings.Contains(v.Key(), "t-") || !strings.HasSuffix(v.Key(), "-"+resolve.Suffix(platform.NativeArch())) {
							continue
						}
						matches, err := runner.MatchesRequirement(selected.Spec, v.Version)
						if err != nil {
							return err
						}
						if !matches {
							continue
						}
						pv, _ := pyver.Parse(v.Version)
						if best == "" || bestVersion.Pre() && !pv.Pre() || bestVersion.Pre() == pv.Pre() && pv.Compare(bestVersion) > 0 {
							best = v.Version
							bestVersion = pv
						}
					}
					if best == "" {
						return errs.New(9, "No available Python matches %s", spec)
					}
					spec = best
				}

				installCmd, _, _ := c.Root().Find([]string{"install"})
				installCmd.SetContext(c.Context())
				o.path = "none"
				if o.dryRun {
					if err := installCmd.ParseFlags([]string{spec}); err != nil {
						return err
					}
					if err := o.plan(installCmd, []string{spec}); err != nil {
						return err
					}
					return o.output(c, map[string]any{"dry_run": true, "reason": selected.Reason, "args": selected.Args}, "Would then run the selected Python with arguments: "+fmt.Sprintf("%q", selected.Args))
				}
				if e = o.install(installCmd, []string{spec}); e != nil {
					return e
				}
				s, e = o.load()
				if e != nil {
					return e
				}
				selected, e = runner.Select(o.root, s, python, args)
			}
		}
		if e != nil {
			return e
		}
		if selected.Dependencies {
			fmt.Fprintln(c.ErrOrStderr(), "pftw run doesn't install dependencies; use a venv or a tool like uv/pipx")
		}
		if o.dryRun {
			return o.output(c, map[string]any{"dry_run": true, "interpreter": selected.Interpreter, "reason": selected.Reason, "argv": append([]string{selected.Interpreter}, selected.Args...)}, fmt.Sprintf("Interpreter: %s\nSelected by: %s\nArgv: %q\nNo files or settings changed; Python was not started.", selected.Interpreter, selected.Reason, append([]string{selected.Interpreter}, selected.Args...)))
		}
		code, e := shims.Launch(selected.Interpreter, selected.Args)
		if e != nil {
			return e
		}
		if code != 0 {
			return &errs.ChildExit{Code: code}
		}
		return nil
	}
	return c
}
