// Command mailbust is a terminal-first disposable mailbox.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/atifrana/mailbust/cli/internal/config"
	"github.com/atifrana/mailbust/cli/internal/mcpserver"
)

// Version is overridden at build time via -ldflags.
var Version = "dev"

// Exit codes (per the project spec): 0 ok, 1 error, 2 timeout, 3 invalid input.
type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

func fail(code int, format string, a ...any) error {
	return exitError{code: code, msg: fmt.Sprintf(format, a...)}
}

func main() {
	err := rootCmd().Execute()
	if err == nil {
		return
	}
	code := 1
	if ee, ok := err.(exitError); ok {
		code = ee.code
	}
	if msg := err.Error(); msg != "" {
		fmt.Fprintln(os.Stderr, "mailbust:", msg)
	}
	os.Exit(code)
}

type opts struct {
	newBox  bool
	code    bool
	link    bool
	list    bool
	jsonOut bool
	timeout int
	ttl     string
}

func rootCmd() *cobra.Command {
	o := &opts{}

	cmd := &cobra.Command{
		Use:   "mailbust [name]",
		Short: "A terminal-first disposable mailbox",
		Long: "MailBust opens a live, disposable inbox in your terminal.\n\n" +
			"Interactive:\n" +
			"  mailbust                 open a random private mailbox\n" +
			"  mailbust testing         open the public named mailbox 'testing'\n\n" +
			"Scripting (no TUI):\n" +
			"  mailbust --new                 create a random mailbox, print its address\n" +
			"  mailbust testing --code        wait for the next code, print it\n" +
			"  mailbust testing --link        wait for the next verify link, print it\n" +
			"  mailbust testing --list        list current messages\n" +
			"  add --json for machine-readable output, --timeout for the wait budget",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return dispatch(config.Load(), o, args)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&o.newBox, "new", false, "create a random mailbox and print its address, then exit")
	f.BoolVar(&o.code, "code", false, "wait for the next verification code, print it, then exit")
	f.BoolVar(&o.link, "link", false, "wait for the next verification link, print it, then exit")
	f.BoolVar(&o.list, "list", false, "print the current messages, then exit")
	f.BoolVar(&o.jsonOut, "json", false, "output JSON instead of plain text")
	f.IntVar(&o.timeout, "timeout", 120, "seconds to wait for --code/--link")
	f.StringVarP(&o.ttl, "time", "t", "", "mailbox lifetime, e.g. 30m, 1h, 24h (default 15m)")

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("mailbust", Version)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:           "mcp",
		Short:         "Run the MCP server over stdio (for AI agents)",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return mcpserver.Run(c.Context(), Version)
		},
	})
	return cmd
}

// dispatch picks interactive vs scripting mode from the flags.
func dispatch(cfg config.Config, o *opts, args []string) error {
	name := ""
	if len(args) == 1 {
		name = args[0]
	}
	if o.code && o.link {
		return fail(3, "--code and --link are mutually exclusive")
	}
	// Validate --time up front so a malformed value is rejected in every mode,
	// even the ones that don't otherwise use it.
	if _, err := parseTTL(o.ttl); err != nil {
		return err
	}

	switch {
	case o.newBox:
		if name != "" {
			return fail(3, "--new does not take a mailbox name")
		}
		return runNew(cfg, o)

	case o.code || o.link:
		kind := "code"
		if o.link {
			kind = "link"
		}
		if name == "" {
			return fail(3, "a mailbox name is required, e.g. mailbust testing --%s", kind)
		}
		return runWait(cfg, name, kind, o)

	case o.list:
		if name == "" {
			return fail(3, "a mailbox name is required, e.g. mailbust testing --list")
		}
		return runList(cfg, name, o)

	default:
		if !isTTY() {
			return fail(3, "non-interactive terminal: use --new, --code, --link, or --list")
		}
		return runTUI(cfg, name, o.ttl)
	}
}
