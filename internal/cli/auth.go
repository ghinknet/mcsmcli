package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"mcsmcli/internal/config"
	"mcsmcli/internal/mcsm"
)

// ---- login ----
// login is special: it runs before PersistentPreRunE so it can create
// the initial config file.  It uses the global persistent flags directly.

var cmdLogin = &cobra.Command{
	Use:   "login",
	Short: "Save panel credentials (URL + API key)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagURL == "" || flagAPIKey == "" {
			return fmt.Errorf("--url and --apikey are required")
		}

		daemonLogin, _ := cmd.Flags().GetString("daemon")

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		name := flagProfile
		if name == "" {
			name = cfg.Current
		}
		if name == "" {
			name = config.DefaultProfileName
		}

		noVerify, _ := cmd.Flags().GetBool("no-verify")
		if !noVerify {
			client := mcsm.New(flagURL, flagAPIKey, flagTimeout)
			ctx, cancel := context.WithTimeout(context.Background(), flagTimeout)
			defer cancel()
			ov, _, err := client.GetOverview(ctx)
			var apiErr *mcsm.APIError
			switch {
			case err == nil:
				fmt.Printf("Connected to panel %s (version %s, nodes %d/%d online)\n",
					flagURL, ov.Version, ov.RemoteCount.Available, ov.RemoteCount.Total)
			case errors.As(err, &apiErr):
				fmt.Printf("Warning: panel reachable but returned %v; saving anyway (non-admin keys may lack overview access)\n", err)
			default:
				return fmt.Errorf("cannot reach panel (use --no-verify to skip): %w", err)
			}
		}

		cfg.Profiles[name] = &config.Profile{URL: flagURL, APIKey: flagAPIKey, Daemon: daemonLogin}
		cfg.Current = name
		if err := cfg.Save(); err != nil {
			return err
		}
		path, _ := config.Path()
		fmt.Printf("✔ Credentials saved to %s (profile: %s)\n", path, name)
		return nil
	},
}

// ---- logout ----

var cmdLogout = &cobra.Command{
	Use:   "logout",
	Short: "Delete saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		name := flagProfile
		if name == "" {
			name = cfg.Current
		}
		if _, ok := cfg.Profiles[name]; !ok {
			return fmt.Errorf("profile %q does not exist", name)
		}
		delete(cfg.Profiles, name)
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("✔ Deleted credentials for profile %q\n", name)
		return nil
	},
}

// ---- whoami ----

var cmdWhoami = &cobra.Command{
	Use:   "whoami",
	Short: "Show current profile and test connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		p := app.profile
		masked := p.APIKey
		if len(masked) > 8 {
			masked = masked[:4] + "****" + masked[len(masked)-4:]
		}
		fmt.Printf("profile:   %s\npanel:     %s\nAPI key:   %s\ndefault daemon: %s\n",
			orDefault(flagProfile, app.cfg.Current), p.URL, masked, orDefault(p.Daemon, "(not set)"))

		ctx, cancel := app.Ctx()
		defer cancel()
		ov, _, err := client.GetOverview(ctx)
		if err != nil {
			fmt.Printf("connectivity: failed (%v)\n", err)
			return nil
		}
		fmt.Printf("connectivity: OK (panel version %s)\n", ov.Version)
		return nil
	},
}

// ---- profile ----

var cmdProfile = &cobra.Command{
	Use:   "profile",
	Short: "Manage multiple panel profiles",
}

var cmdProfileList = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		w := newTable()
		fmt.Fprintln(w, "  current\tname\tpanel URL\tdefault daemon")
		for name, p := range cfg.Profiles {
			mark := " "
			if name == cfg.Current {
				mark = "*"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", mark, name, p.URL, p.Daemon)
		}
		return w.Flush()
	},
}

var cmdProfileUse = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := cfg.Profiles[args[0]]; !ok {
			return fmt.Errorf("profile %q does not exist", args[0])
		}
		cfg.Current = args[0]
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("✔ Switched to profile %q\n", args[0])
		return nil
	},
}

var cmdProfileSetDaemon = &cobra.Command{
	Use:   "set-daemon <daemonId>",
	Short: "Set default daemon for the current profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		name := orDefault(flagProfile, cfg.Current)
		p, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("profile %q does not exist; run mcsmcli login first", name)
		}
		p.Daemon = args[0]
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("✔ Default daemon for profile %q set to %s\n", name, args[0])
		return nil
	},
}

func init() {
	// login has its own --daemon flag (separate from global -d) and --no-verify.
	cmdLogin.Flags().String("daemon", "", "default daemonId (optional)")
	cmdLogin.Flags().Bool("no-verify", false, "skip connectivity check")

	cmdProfile.AddCommand(cmdProfileList)
	cmdProfile.AddCommand(cmdProfileUse)
	cmdProfile.AddCommand(cmdProfileSetDaemon)
}

// Ensure time import is used (it is referenced above).
var _ = time.Now
