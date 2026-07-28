package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"mcsmcli/internal/mcsm"
)

// ---- user ----

var cmdUser = &cobra.Command{
	Use:   "user",
	Short: "Manage panel users",
}

var cmdUserList = &cobra.Command{
	Use:   "list",
	Short: "Search / list users",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		role, _ := cmd.Flags().GetString("role")
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")

		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		pageData, raw, err := client.ListUsers(ctx, name, role, page, size)
		if err != nil {
			return err
		}
		if flagJSON {
			app.PrintRaw(raw)
			return nil
		}
		w := newTable()
		fmt.Fprintln(w, "  uuid\tusername\tpermission\t2FA\tinstances\tlast login")
		for _, u := range pageData.Data {
			fmt.Fprintf(w, "  %s\t%s\t%s\t%v\t%d\t%s\n",
				u.UUID, u.UserName, mcsm.PermissionText(u.Permission),
				u.Open2FA, len(u.Instances), u.LoginTime)
		}
		w.Flush()
		fmt.Printf("  %d users total, page %d/%d\n", pageData.Total, pageData.Page, pageData.MaxPage)
		return nil
	},
}

var cmdUserCreate = &cobra.Command{
	Use:   "create",
	Short: "Create a user",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		password, _ := cmd.Flags().GetString("password")
		permission, _ := cmd.Flags().GetInt("permission")
		if name == "" || password == "" {
			return fmt.Errorf("--name and --password are required")
		}
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		raw, err := client.CreateUser(ctx, name, password, permission)
		if err != nil {
			return err
		}
		app.PrintRaw(raw)
		return nil
	},
}

var cmdUserUpdate = &cobra.Command{
	Use:   "update <uuid>",
	Short: "Update user info",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		inline, _ := cmd.Flags().GetString("config")
		permission, _ := cmd.Flags().GetInt("permission")

		var body any
		if file != "" || inline != "" {
			var err error
			if body, err = loadBodyJSON(file, inline); err != nil {
				return err
			}
		} else if cmd.Flags().Changed("permission") {
			body = map[string]any{"permission": permission}
		} else {
			return fmt.Errorf("no update fields provided")
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.UpdateUser(ctx, args[0], body); err != nil {
			return err
		}
		app.OK("User %s updated", args[0])
		return nil
	},
}

var cmdUserDelete = &cobra.Command{
	Use:   "delete <uuid>...",
	Short: "Delete users",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: mcsmcli user delete <uuid>...")
		}
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.DeleteUsers(ctx, args); err != nil {
			return err
		}
		app.OK("Deleted %d user(s)", len(args))
		return nil
	},
}

func init() {
	cmdUserList.Flags().String("name", "", "filter by username")
	cmdUserList.Flags().String("role", "", "filter by role: 1=user 10=admin -1=banned")
	cmdUserList.Flags().Int("page", 1, "page number")
	cmdUserList.Flags().Int("size", 20, "page size")

	cmdUserCreate.Flags().String("name", "", "username")
	cmdUserCreate.Flags().String("password", "", "password")
	cmdUserCreate.Flags().Int("permission", 1, "permission: 1=user 10=admin -1=banned")

	cmdUserUpdate.Flags().String("file", "", "user config JSON file path (- for stdin)")
	cmdUserUpdate.Flags().String("config", "", "inline user config JSON")
	cmdUserUpdate.Flags().Int("permission", 1, "permission: 1=user 10=admin -1=banned")

	cmdUser.AddCommand(cmdUserList)
	cmdUser.AddCommand(cmdUserCreate)
	cmdUser.AddCommand(cmdUserUpdate)
	cmdUser.AddCommand(cmdUserDelete)
}
