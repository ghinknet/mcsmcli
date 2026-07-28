package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"mcsmcli/internal/mcsm"
)

// newPowerCmd creates a start/stop/restart/kill command.
func newPowerCmd(name, apiOp, verb string) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s <uuid>", name),
		Short: fmt.Sprintf("%s an instance", verb),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			daemonID, err := getDaemonID(cmd)
			if err != nil {
				return err
			}
			app := getApp(cmd)
			ctx, cancel := app.Ctx()
			defer cancel()
			if err := client.InstancePower(ctx, daemonID, args[0], apiOp); err != nil {
				return err
			}
			app.OK("Instance %s %s command sent", args[0], verb)
			return nil
		},
	}
}

// newBatchCmd creates a batch-start/stop/restart/kill command.
func newBatchCmd(op, verb string) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("batch-%s <uuid[@daemonId]>...", op),
		Short: fmt.Sprintf("Batch %s instances", verb),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			app := getApp(cmd)
			refs, err := refsFromArgs(app, args)
			if err != nil {
				return err
			}
			ctx, cancel := app.Ctx()
			defer cancel()
			if err := client.BatchInstanceOp(ctx, op, refs); err != nil {
				return err
			}
			app.OK("Batch %s sent to %d instances", verb, len(refs))
			return nil
		},
	}
}

// ---- instance ----

var cmdInstance = &cobra.Command{
	Use:   "instance",
	Short: "Manage instances (start/stop, create/delete, commands, logs, etc.)",
}

var cmdInstanceList = &cobra.Command{
	Use:   "list",
	Short: "List instances on a daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		name, _ := cmd.Flags().GetString("name")
		status, _ := cmd.Flags().GetString("status")

		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		pageData, raw, err := client.ListInstances(ctx, daemonID, page, size, name, status)
		if err != nil {
			return err
		}
		if flagJSON {
			app.PrintRaw(raw)
			return nil
		}
		w := newTable()
		fmt.Fprintln(w, "  uuid\tname\tstatus\ttype\tstart count\texpires")
		for _, ins := range pageData.Data {
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%d\t%s\n",
				ins.InstanceUUID, ins.Config.Nickname, mcsm.StatusText(ins.Status),
				ins.Config.Type, ins.Started, fmtMillis(ins.Config.EndTime))
		}
		w.Flush()
		fmt.Printf("  page %d/%d\n", page, pageData.MaxPage)
		return nil
	},
}

var cmdInstanceInfo = &cobra.Command{
	Use:   "info <uuid>",
	Short: "Show instance details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		ins, raw, err := client.GetInstance(ctx, daemonID, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			app.PrintRaw(raw)
			return nil
		}
		c := ins.Config
		fmt.Printf("Instance:    %s (%s)\n", c.Nickname, ins.InstanceUUID)
		fmt.Printf("Status:      %s (started %d times)\n", mcsm.StatusText(ins.Status), ins.Started)
		fmt.Printf("Type:        %s / %s\n", c.Type, orDefault(c.ProcessType, "general"))
		fmt.Printf("Working dir: %s\n", c.Cwd)
		fmt.Printf("Start cmd:   %s\n", c.StartCommand)
		fmt.Printf("Stop cmd:    %s\n", c.StopCommand)
		fmt.Printf("Created:     %s  Last start: %s  Expires: %s\n",
			fmtMillis(c.CreateDatetime), fmtMillis(c.LastDatetime), fmtMillis(c.EndTime))
		fmt.Printf("Disk usage:  %s\n", fmtBytes(ins.Space))
		if ins.Status == mcsm.StatusRunning {
			fmt.Printf("Process:     PID %d, CPU %s, Memory %s\n",
				ins.ProcessInfo.PID, fmtPercent(ins.ProcessInfo.CPU), fmtBytes(ins.ProcessInfo.Memory))
		}
		if ins.Info.MaxPlayers > 0 {
			fmt.Printf("Players:     %d/%d\n", ins.Info.CurrentPlayers, ins.Info.MaxPlayers)
		}
		return nil
	},
}

var cmdInstanceCreate = &cobra.Command{
	Use:   "create",
	Short: "Create an instance (config from JSON)",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		inline, _ := cmd.Flags().GetString("config")
		body, err := loadBodyJSON(file, inline)
		if err != nil {
			return err
		}
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		raw, err := client.CreateInstance(ctx, daemonID, body)
		if err != nil {
			return err
		}
		app.PrintRaw(raw)
		return nil
	},
}

var cmdInstanceUpdate = &cobra.Command{
	Use:   "update <uuid>",
	Short: "Update instance config (JSON or convenience flags)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		inline, _ := cmd.Flags().GetString("config")
		nickname, _ := cmd.Flags().GetString("nickname")
		startCmd, _ := cmd.Flags().GetString("start-cmd")
		stopCmd, _ := cmd.Flags().GetString("stop-cmd")
		cwd, _ := cmd.Flags().GetString("cwd")

		var body any
		if file != "" || inline != "" {
			var err error
			if body, err = loadBodyJSON(file, inline); err != nil {
				return err
			}
		} else {
			cfg := map[string]any{}
			for k, v := range map[string]string{
				"nickname": nickname, "startCommand": startCmd, "stopCommand": stopCmd, "cwd": cwd,
			} {
				if v != "" {
					cfg[k] = v
				}
			}
			if len(cfg) == 0 {
				return fmt.Errorf("no update fields provided")
			}
			body = cfg
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if _, err := client.UpdateInstanceConfig(ctx, daemonID, args[0], body); err != nil {
			return err
		}
		app.OK("Instance %s config updated", args[0])
		return nil
	},
}

var cmdInstanceDelete = &cobra.Command{
	Use:   "delete <uuid>...",
	Short: "Delete instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: mcsmcli instance delete <uuid>... [--files]")
		}
		deleteFiles, _ := cmd.Flags().GetBool("files")
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if _, err := client.DeleteInstances(ctx, daemonID, args, deleteFiles); err != nil {
			return err
		}
		app.OK("Deleted %d instance(s) (delete files: %v)", len(args), deleteFiles)
		return nil
	},
}

var cmdInstanceCmd = &cobra.Command{
	Use:   "cmd <uuid> <command...>",
	Short: "Send a command to an instance",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		command := strings.Join(args[1:], " ")
		if err := client.SendCommand(ctx, daemonID, args[0], command); err != nil {
			return err
		}
		app.OK("Command sent: %s", command)
		return nil
	},
}

var cmdInstanceLog = &cobra.Command{
	Use:   "log <uuid>",
	Short: "View instance terminal output",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		size, _ := cmd.Flags().GetInt("size")
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		logText, err := client.GetOutputLog(ctx, daemonID, args[0], size)
		if err != nil {
			return err
		}
		fmt.Print(logText)
		if !strings.HasSuffix(logText, "\n") {
			fmt.Println()
		}
		return nil
	},
}

var cmdInstanceUpgrade = &cobra.Command{
	Use:   "upgrade <uuid>",
	Short: "Trigger instance update task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.UpgradeInstance(ctx, daemonID, args[0]); err != nil {
			return err
		}
		app.OK("Instance %s update task triggered", args[0])
		return nil
	},
}

var cmdInstanceReinstall = &cobra.Command{
	Use:   "reinstall <uuid>",
	Short: "Reinstall an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetURL, _ := cmd.Flags().GetString("target-url")
		title, _ := cmd.Flags().GetString("title")
		desc, _ := cmd.Flags().GetString("desc")
		if targetURL == "" {
			return fmt.Errorf("--target-url is required")
		}
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		daemonID, err := getDaemonID(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.ReinstallInstance(ctx, daemonID, args[0], targetURL, title, desc); err != nil {
			return err
		}
		app.OK("Instance %s reinstall started", args[0])
		return nil
	},
}

func init() {
	// Instance list flags.
	cmdInstanceList.Flags().Int("page", 1, "page number")
	cmdInstanceList.Flags().Int("size", 20, "page size")
	cmdInstanceList.Flags().String("name", "", "filter by instance name")
	cmdInstanceList.Flags().String("status", "", "filter by status")

	// Instance create flags.
	cmdInstanceCreate.Flags().String("file", "", "InstanceConfig JSON file path (- for stdin)")
	cmdInstanceCreate.Flags().String("config", "", "inline InstanceConfig JSON")

	// Instance update flags.
	cmdInstanceUpdate.Flags().String("file", "", "InstanceConfig JSON file path (- for stdin)")
	cmdInstanceUpdate.Flags().String("config", "", "inline InstanceConfig JSON")
	cmdInstanceUpdate.Flags().String("nickname", "", "instance name")
	cmdInstanceUpdate.Flags().String("start-cmd", "", "start command")
	cmdInstanceUpdate.Flags().String("stop-cmd", "", "stop command")
	cmdInstanceUpdate.Flags().String("cwd", "", "working directory")

	// Instance delete flags.
	cmdInstanceDelete.Flags().Bool("files", false, "also delete instance files (irreversible)")

	// Instance log flags.
	cmdInstanceLog.Flags().Int("size", 0, "log size in KB (1-2048, 0 for all)")

	// Instance reinstall flags.
	cmdInstanceReinstall.Flags().String("target-url", "", "install package URL")
	cmdInstanceReinstall.Flags().String("title", "", "title")
	cmdInstanceReinstall.Flags().String("desc", "", "description")

	cmdInstance.AddCommand(cmdInstanceList)
	cmdInstance.AddCommand(cmdInstanceInfo)
	cmdInstance.AddCommand(newPowerCmd("start", "open", "start"))
	cmdInstance.AddCommand(newPowerCmd("stop", "stop", "stop"))
	cmdInstance.AddCommand(newPowerCmd("restart", "restart", "restart"))
	cmdInstance.AddCommand(newPowerCmd("kill", "kill", "kill"))
	cmdInstance.AddCommand(newBatchCmd("start", "start"))
	cmdInstance.AddCommand(newBatchCmd("stop", "stop"))
	cmdInstance.AddCommand(newBatchCmd("restart", "restart"))
	cmdInstance.AddCommand(newBatchCmd("kill", "kill"))
	cmdInstance.AddCommand(cmdInstanceCreate)
	cmdInstance.AddCommand(cmdInstanceUpdate)
	cmdInstance.AddCommand(cmdInstanceDelete)
	cmdInstance.AddCommand(cmdInstanceCmd)
	cmdInstance.AddCommand(cmdInstanceLog)
	cmdInstance.AddCommand(cmdInstanceUpgrade)
	cmdInstance.AddCommand(cmdInstanceReinstall)
}
