package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ---- image ----

var cmdImage = &cobra.Command{
	Use:   "image",
	Short: "Manage Docker images, containers, and networks",
}

var cmdImageList = &cobra.Command{
	Use:   "list",
	Short: "List Docker images",
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
		raw, err := client.ListImages(ctx, daemonID)
		if err != nil {
			return err
		}
		app.PrintRaw(raw)
		return nil
	},
}

var cmdImageContainers = &cobra.Command{
	Use:   "containers",
	Short: "List Docker containers",
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
		raw, err := client.ListContainers(ctx, daemonID)
		if err != nil {
			return err
		}
		app.PrintRaw(raw)
		return nil
	},
}

var cmdImageNetworks = &cobra.Command{
	Use:   "networks",
	Short: "List Docker networks",
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
		raw, err := client.ListNetworks(ctx, daemonID)
		if err != nil {
			return err
		}
		app.PrintRaw(raw)
		return nil
	},
}

var cmdImageBuild = &cobra.Command{
	Use:   "build",
	Short: "Build a Docker image from a Dockerfile",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		tag, _ := cmd.Flags().GetString("tag")
		dockerfile, _ := cmd.Flags().GetString("dockerfile")
		if name == "" || dockerfile == "" {
			return fmt.Errorf("--name and --dockerfile are required")
		}
		var content []byte
		var err error
		if dockerfile == "-" {
			content, err = readAllStdin()
		} else {
			content, err = os.ReadFile(dockerfile)
		}
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
		if err := client.BuildImage(ctx, daemonID, string(content), name, tag); err != nil {
			return err
		}
		app.OK("Image %s:%s build submitted; check progress with mcsmcli image progress", name, tag)
		return nil
	},
}

var cmdImageProgress = &cobra.Command{
	Use:   "progress",
	Short: "Query image build progress",
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
		progress, err := client.BuildProgress(ctx, daemonID)
		if err != nil {
			return err
		}
		if len(progress) == 0 {
			fmt.Println("No build tasks")
			return nil
		}
		statusText := map[int]string{-1: "failed", 1: "building", 2: "complete"}
		w := newTable()
		fmt.Fprintln(w, "  image\tstatus")
		for _, name := range sortedKeys(progress) {
			fmt.Fprintf(w, "  %s\t%s\n", name, orDefault(statusText[progress[name]], "unknown"))
		}
		return w.Flush()
	},
}

func init() {
	cmdImageBuild.Flags().String("name", "", "image name")
	cmdImageBuild.Flags().String("tag", "latest", "version tag")
	cmdImageBuild.Flags().String("dockerfile", "", "Dockerfile path (- for stdin)")

	cmdImage.AddCommand(cmdImageList)
	cmdImage.AddCommand(cmdImageContainers)
	cmdImage.AddCommand(cmdImageNetworks)
	cmdImage.AddCommand(cmdImageBuild)
	cmdImage.AddCommand(cmdImageProgress)
}
