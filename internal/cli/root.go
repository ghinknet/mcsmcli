// Package cli implements the mcsmcli command-line interface using Cobra.
package cli

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"go.gh.ink/timex"
	"go.gh.ink/toolbox/expr"

	"mcsmcli/internal/config"
	"mcsmcli/internal/mcsm"
)

// Global flag values (bound via PersistentFlags on root).
var (
	flagProfile string
	flagURL     string
	flagAPIKey  string
	flagDaemon  string
	flagJSON    bool
	flagTimeout time.Duration
)

// App holds runtime state shared across commands.
type App struct {
	cfg     *config.Config
	profile *config.Profile
}

// newApp creates an App by loading the config file.
func newApp() (*App, error) {
	// Ensure viper is set up with config paths and env bindings.
	// Init is idempotent — safe to call on every command invocation.
	if err := config.Init(); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &App{cfg: cfg}, nil
}

// Client builds an API client from the current flag/profile state.
func (a *App) Client() (*mcsm.Client, error) {
	p, _, err := a.cfg.Resolve(flagProfile, flagURL, flagAPIKey)
	if err != nil {
		return nil, err
	}
	a.profile = p
	return mcsm.New(p.URL, p.APIKey, flagTimeout), nil
}

// DaemonID resolves the effective daemonId: -d flag > env/profile default.
func (a *App) DaemonID() (string, error) {
	if flagDaemon != "" {
		return flagDaemon, nil
	}
	if a.profile != nil && a.profile.Daemon != "" {
		return a.profile.Daemon, nil
	}
	return "", fmt.Errorf("no daemon specified: use -d <daemonId> or set a default with mcsmcli profile set-daemon")
}

// Ctx returns a context with the global timeout.
func (a *App) Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), flagTimeout)
}

// PrintRaw pretty-prints raw JSON data to stdout.
func (a *App) PrintRaw(raw mcsm.RawMessage) {
	var buf bytes.Buffer
	if stdjson.Indent(&buf, raw, "", "  ") != nil {
		os.Stdout.Write(raw)
		fmt.Println()
		return
	}
	fmt.Println(buf.String())
}

// OK prints a success message.  In --json mode it prints {"ok":true}.
func (a *App) OK(format string, args ...any) {
	if flagJSON {
		fmt.Println(`{"ok": true}`)
		return
	}
	fmt.Printf("✔ "+format+"\n", args...)
}

// ---- root command ----

// Execute is the CLI entry point.  It returns 0 on success, 1 on error.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}

var rootCmd = &cobra.Command{
	Use:   "mcsmcli",
	Short: "MCSManager command-line tool",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config load for login (it saves config, not reads it).
		if cmd.Name() == "login" {
			return nil
		}
		app, err := newApp()
		if err != nil {
			return err
		}
		cmd.SetContext(context.WithValue(cmd.Context(), ctxKeyApp, app))
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ctxKeyApp is the context key for the App pointer.
type ctxKey int

const ctxKeyApp ctxKey = iota

// getApp retrieves the App from a cobra command's context.
func getApp(cmd *cobra.Command) *App {
	// Walk up the command chain to find the App.
	for c := cmd; c != nil; c = c.Parent() {
		if ctx := c.Context(); ctx != nil {
			if a, ok := ctx.Value(ctxKeyApp).(*App); ok {
				return a
			}
		}
	}
	// Should never happen if PersistentPreRunE ran correctly.
	panic("App not initialized—PersistentPreRunE may have been skipped")
}

// getClient is a convenience wrapper.
func getClient(cmd *cobra.Command) (*mcsm.Client, error) {
	return getApp(cmd).Client()
}

// getDaemonID is a convenience wrapper.
func getDaemonID(cmd *cobra.Command) (string, error) {
	return getApp(cmd).DaemonID()
}

func init() {
	// Global persistent flags.
	rootCmd.PersistentFlags().StringVar(&flagProfile, "profile", "", "profile name (defaults to current)")
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "override panel URL")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "apikey", "", "override API key")
	rootCmd.PersistentFlags().StringVarP(&flagDaemon, "daemon", "d", "", "daemonId (node ID)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output raw JSON data")
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second, "HTTP request timeout")

	// Register subcommands (defined in sibling files).
	rootCmd.AddCommand(cmdLogin)
	rootCmd.AddCommand(cmdLogout)
	rootCmd.AddCommand(cmdWhoami)
	rootCmd.AddCommand(cmdProfile)
	rootCmd.AddCommand(cmdOverview)
	rootCmd.AddCommand(cmdDaemon)
	rootCmd.AddCommand(cmdInstance)
	rootCmd.AddCommand(cmdUser)
	rootCmd.AddCommand(cmdFile)
	rootCmd.AddCommand(cmdImage)
}

// ---- output formatting helpers ----

func fmtBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, e := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		e++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[e])
}

func fmtMillis(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	return timex.UnixMilli(ms).Local().Format("2006-01-02 15:04:05")
}

func fmtPercent(f float64) string {
	// Panel fields are either 0–1 ratio or 0–100 percentage; normalise by order of magnitude.
	return fmt.Sprintf("%.1f%%", expr.Ternary(f <= 1, f*100, f))
}

func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
}

// loadBodyJSON reads a request body from --file ("-" for stdin) or an inline JSON string.
func loadBodyJSON(file, inline string) (mcsm.RawMessage, error) {
	switch {
	case file == "-":
		return readAllStdin()
	case file != "":
		return os.ReadFile(file)
	case inline != "":
		return mcsm.RawMessage(inline), nil
	}
	return nil, fmt.Errorf("provide config via --file <path|-> or --config '<json>'")
}

func readAllStdin() ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(os.Stdin); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// refsFromArgs parses batch-operation arguments: each is "uuid" or "uuid@daemonId".
func refsFromArgs(a *App, args []string) ([]mcsm.InstanceRef, error) {
	refs := make([]mcsm.InstanceRef, 0, len(args))
	for _, arg := range args {
		uuid, daemon, found := strings.Cut(arg, "@")
		if !found {
			var err error
			if daemon, err = a.DaemonID(); err != nil {
				return nil, fmt.Errorf("%q missing @daemonId and no default daemon: %w", arg, err)
			}
		}
		refs = append(refs, mcsm.InstanceRef{InstanceUUID: uuid, DaemonID: daemon})
	}
	return refs, nil
}
