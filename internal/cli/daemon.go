package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"go.gh.ink/toolbox/xtype"

	"mcsmcli/internal/mcsm"
)

// ---- overview ----

var cmdOverview = &cobra.Command{
	Use:   "overview",
	Short: "Show panel overview (version, system, login records, node stats)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		ov, raw, err := client.GetOverview(ctx)
		if err != nil {
			return err
		}
		if flagJSON {
			app.PrintRaw(raw)
			return nil
		}
		fmt.Printf("Panel version: %s (daemon spec %s)\n", ov.Version, ov.SpecifiedDaemonVersion)
		fmt.Printf("System:        %s %s (%s, Node %s)\n", ov.System.Type, ov.System.Release, ov.System.Hostname, ov.System.Node)
		fmt.Printf("Panel CPU:     %s  Memory: %s / %s available\n",
			fmtPercent(ov.System.CPU), fmtBytes(ov.System.FreeMem), fmtBytes(ov.System.TotalMem))
		fmt.Printf("Logins:        %d success, %d failed, %d illegal, %d banned IPs\n",
			ov.Record.Logined, ov.Record.LoginFailed, ov.Record.IllegalAccess, ov.Record.BanIPs)
		fmt.Printf("Nodes:         %d/%d online\n", ov.RemoteCount.Available, ov.RemoteCount.Total)
		return nil
	},
}

// ---- daemon ----

var cmdDaemon = &cobra.Command{
	Use:   "daemon",
	Short: "Manage daemons (nodes)",
}

var cmdDaemonList = &cobra.Command{
	Use:   "list",
	Short: "List all daemons",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		ov, raw, err := client.GetOverview(ctx)
		if err != nil {
			return err
		}
		if flagJSON {
			app.PrintRaw(raw)
			return nil
		}
		w := newTable()
		fmt.Fprintln(w, "  daemonId\tremarks\taddress\tstatus\tversion\tinstances\tCPU\tMemory")
		for _, d := range ov.Remote {
			status := "offline"
			if d.Available {
				status = "online"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s:%d%s\t%s\t%s\t%d/%d\t%s\t%s\n",
				d.UUID, d.Remarks, d.IP, d.Port, d.Prefix, status, d.Version,
				d.Instance.Running, d.Instance.Total,
				fmtPercent(d.System.CPUUsage), fmtPercent(d.System.MemUsage))
		}
		return w.Flush()
	},
}

var cmdDaemonSystems = &cobra.Command{
	Use:   "systems",
	Short: "List daemon system info (remote_services_system)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		raw, err := client.ListDaemonsSystem(ctx)
		if err != nil {
			return err
		}
		app.PrintRaw(raw)
		return nil
	},
}

var cmdDaemonAdd = &cobra.Command{
	Use:   "add",
	Short: "Add a new daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, _ := cmd.Flags().GetString("ip")
		if ip == "" {
			return fmt.Errorf("--ip is required")
		}
		port, _ := cmd.Flags().GetInt("port")
		prefix, _ := cmd.Flags().GetString("prefix")
		remarks, _ := cmd.Flags().GetString("remarks")
		key, _ := cmd.Flags().GetString("key")

		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		id, err := client.AddDaemon(ctx, ip, port, prefix, remarks, key)
		if err != nil {
			return err
		}
		app.OK("Daemon added, daemonId: %s", id)
		return nil
	},
}

var cmdDaemonDelete = &cobra.Command{
	Use:   "delete <daemonId>",
	Short: "Delete a daemon",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.DeleteDaemon(ctx, args[0]); err != nil {
			return err
		}
		app.OK("Daemon %s deleted", args[0])
		return nil
	},
}

var cmdDaemonLink = &cobra.Command{
	Use:   "link <daemonId>",
	Short: "Try to reconnect a daemon",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.LinkDaemon(ctx, args[0]); err != nil {
			return err
		}
		app.OK("Daemon %s connected successfully", args[0])
		return nil
	},
}

var cmdDaemonUpdate = &cobra.Command{
	Use:   "update <daemonId>",
	Short: "Update daemon connection config (unspecified fields keep current values)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		daemonID := args[0]
		ip, _ := cmd.Flags().GetString("ip")
		port, _ := cmd.Flags().GetInt("port")
		prefix, _ := cmd.Flags().GetString("prefix")
		remarks, _ := cmd.Flags().GetString("remarks")
		key, _ := cmd.Flags().GetString("key")
		available, _ := cmd.Flags().GetString("available")

		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()

		// API requires full config; fetch current state first, then merge.
		ov, _, err := client.GetOverview(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch current daemon config: %w", err)
		}
		var cur *mcsm.Daemon
		for i := range ov.Remote {
			if ov.Remote[i].UUID == daemonID {
				cur = &ov.Remote[i]
				break
			}
		}
		if cur == nil {
			return fmt.Errorf("daemon %s not found", daemonID)
		}

		body := xtype.H{
			"uuid":      daemonID,
			"ip":        orDefault(ip, cur.IP),
			"port":      cur.Port,
			"prefix":    cur.Prefix,
			"available": cur.Available,
			"remarks":   cur.Remarks,
			"apiKey":    "",
		}
		if port != 0 {
			body["port"] = port
		}
		if cmd.Flags().Changed("prefix") {
			body["prefix"] = prefix
		}
		if cmd.Flags().Changed("remarks") {
			body["remarks"] = remarks
		}
		if cmd.Flags().Changed("key") {
			body["apiKey"] = key
		}
		if available != "" {
			b, err := strconv.ParseBool(available)
			if err != nil {
				return fmt.Errorf("--available must be true or false")
			}
			body["available"] = b
		}
		if err := client.UpdateDaemon(ctx, daemonID, body); err != nil {
			return err
		}
		app.OK("Daemon %s config updated", daemonID)
		return nil
	},
}

func init() {
	cmdDaemonAdd.Flags().String("ip", "", "daemon address")
	cmdDaemonAdd.Flags().Int("port", 24444, "daemon port")
	cmdDaemonAdd.Flags().String("prefix", "", "path prefix")
	cmdDaemonAdd.Flags().String("remarks", "", "remarks")
	cmdDaemonAdd.Flags().String("key", "", "daemon API key")

	cmdDaemonUpdate.Flags().String("ip", "", "daemon address")
	cmdDaemonUpdate.Flags().Int("port", 0, "daemon port")
	cmdDaemonUpdate.Flags().String("prefix", "", "path prefix")
	cmdDaemonUpdate.Flags().String("remarks", "", "remarks")
	cmdDaemonUpdate.Flags().String("key", "", "daemon API key")
	cmdDaemonUpdate.Flags().String("available", "", "enable/disable (true/false)")

	cmdDaemon.AddCommand(cmdDaemonList)
	cmdDaemon.AddCommand(cmdDaemonSystems)
	cmdDaemon.AddCommand(cmdDaemonAdd)
	cmdDaemon.AddCommand(cmdDaemonDelete)
	cmdDaemon.AddCommand(cmdDaemonLink)
	cmdDaemon.AddCommand(cmdDaemonUpdate)
}
