package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.gh.ink/toolbox/expr"

	"mcsmcli/internal/mcsm"
)

// ---- file ----

var cmdFile = &cobra.Command{
	Use:   "file",
	Short: "Manage instance files",
}

var cmdFileLs = &cobra.Command{
	Use:   "ls <uuid> [path]",
	Short: "List directory contents",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		target := "/"
		if len(args) == 2 {
			target = args[1]
		}
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		pageData, raw, err := client.ListFiles(ctx, daemonID, args[0], target, page, size)
		if err != nil {
			return err
		}
		if flagJSON {
			app.PrintRaw(raw)
			return nil
		}
		w := newTable()
		fmt.Fprintln(w, "  type\tmode\tsize\tmodified\tname")
		for _, item := range pageData.Items {
			fmt.Fprintf(w, "  %s\t%d\t%s\t%s\t%s\n",
				expr.Ternary(item.Type == 0, "dir", "file"),
				item.Mode, fmtBytes(item.Size), item.Time, item.Name)
		}
		w.Flush()
		fmt.Printf("  %s %d item(s)\n", pageData.AbsolutePath, pageData.Total)
		return nil
	},
}

var cmdFileCat = &cobra.Command{
	Use:   "cat <uuid> <file-path>",
	Short: "View file contents",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		content, err := client.ReadFile(ctx, daemonID, args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Print(content)
		return nil
	},
}

var cmdFileWrite = &cobra.Command{
	Use:   "write <uuid> <file-path>",
	Short: "Write file contents",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		text, _ := cmd.Flags().GetString("text")
		from, _ := cmd.Flags().GetString("from")
		if from != "" {
			raw, err := loadBodyJSON(from, "")
			if err != nil {
				return err
			}
			text = string(raw)
		} else if text == "" {
			return fmt.Errorf("provide content via --text or --from")
		}
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.WriteFile(ctx, daemonID, args[0], args[1], text); err != nil {
			return err
		}
		app.OK("File %s written", args[1])
		return nil
	},
}

var cmdFileDownload = &cobra.Command{
	Use:   "download <uuid> <remote-path> [local-path]",
	Short: "Download a file from an instance",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		local := expr.Ternary(len(args) == 3, args[2], baseName(args[1]))
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		// Large file transfer; no timeout.
		if err := client.DownloadFile(context.Background(), daemonID, args[0], args[1], local); err != nil {
			return err
		}
		app.OK("Downloaded %s -> %s", args[1], local)
		return nil
	},
}

var cmdFileUpload = &cobra.Command{
	Use:   "upload <uuid> <local-file> [remote-dir]",
	Short: "Upload a local file to an instance",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := expr.Ternary(len(args) == 3, args[2], "/")
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		// Large file transfer; no timeout.
		if err := client.UploadFile(context.Background(), daemonID, args[0], dir, args[1]); err != nil {
			return err
		}
		app.OK("Uploaded %s -> %s", args[1], dir)
		return nil
	},
}

var cmdFileCp = &cobra.Command{
	Use:   "cp <uuid> <src> <dst> [<src> <dst>...]",
	Short: "Copy files",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		pairs, err := pairTargets(args[1:])
		if err != nil {
			return err
		}
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.CopyFiles(ctx, daemonID, args[0], pairs); err != nil {
			return err
		}
		app.OK("Copied %d item(s)", len(pairs))
		return nil
	},
}

var cmdFileMv = &cobra.Command{
	Use:   "mv <uuid> <src> <dst> [<src> <dst>...]",
	Short: "Move / rename files",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		pairs, err := pairTargets(args[1:])
		if err != nil {
			return err
		}
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.MoveFiles(ctx, daemonID, args[0], pairs); err != nil {
			return err
		}
		app.OK("Moved %d item(s)", len(pairs))
		return nil
	},
}

var cmdFileZip = &cobra.Command{
	Use:   "zip <uuid> <dest-zip> <target>...",
	Short: "Compress files (utf-8)",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.CompressFiles(ctx, daemonID, args[0], args[1], args[2:]); err != nil {
			return err
		}
		app.OK("Compress task submitted: %s", args[1])
		return nil
	},
}

var cmdFileUnzip = &cobra.Command{
	Use:   "unzip <uuid> <zip-path> <dest-dir>",
	Short: "Decompress a file",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		code, _ := cmd.Flags().GetString("code")
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.DecompressFile(ctx, daemonID, args[0], args[1], args[2], code); err != nil {
			return err
		}
		app.OK("Decompress task submitted: %s -> %s", args[1], args[2])
		return nil
	},
}

var cmdFileRm = &cobra.Command{
	Use:   "rm <uuid> <path>...",
	Short: "Delete files / directories",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.DeleteFiles(ctx, daemonID, args[0], args[1:]); err != nil {
			return err
		}
		app.OK("Deleted %d item(s)", len(args)-1)
		return nil
	},
}

var cmdFileTouch = &cobra.Command{
	Use:   "touch <uuid> <path>",
	Short: "Create an empty file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.TouchFile(ctx, daemonID, args[0], args[1]); err != nil {
			return err
		}
		app.OK("File created: %s", args[1])
		return nil
	},
}

var cmdFileMkdir = &cobra.Command{
	Use:   "mkdir <uuid> <path>",
	Short: "Create a directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, daemonID, err := fileCtx(cmd)
		if err != nil {
			return err
		}
		app := getApp(cmd)
		ctx, cancel := app.Ctx()
		defer cancel()
		if err := client.Mkdir(ctx, daemonID, args[0], args[1]); err != nil {
			return err
		}
		app.OK("Directory created: %s", args[1])
		return nil
	},
}

// ---- helpers ----

func fileCtx(cmd *cobra.Command) (*mcsm.Client, string, error) {
	client, err := getClient(cmd)
	if err != nil {
		return nil, "", err
	}
	daemonID, err := getDaemonID(cmd)
	if err != nil {
		return nil, "", err
	}
	return client, daemonID, nil
}

// pairTargets splits an even-length list into [src, dst] pairs.
func pairTargets(args []string) ([][2]string, error) {
	if len(args) == 0 || len(args)%2 != 0 {
		return nil, fmt.Errorf("arguments must be paired: <src> <dst>")
	}
	pairs := make([][2]string, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		pairs = append(pairs, [2]string{args[i], args[i+1]})
	}
	return pairs, nil
}

func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}

func init() {
	cmdFileLs.Flags().Int("page", 0, "page number (0-based)")
	cmdFileLs.Flags().Int("size", 100, "page size")

	cmdFileWrite.Flags().String("text", "", "text content to write")
	cmdFileWrite.Flags().String("from", "", "read content from local file (- for stdin)")

	cmdFileUnzip.Flags().String("code", "utf-8", "archive encoding: utf-8, gbk, big5")

	cmdFile.AddCommand(cmdFileLs)
	cmdFile.AddCommand(cmdFileCat)
	cmdFile.AddCommand(cmdFileWrite)
	cmdFile.AddCommand(cmdFileDownload)
	cmdFile.AddCommand(cmdFileUpload)
	cmdFile.AddCommand(cmdFileCp)
	cmdFile.AddCommand(cmdFileMv)
	cmdFile.AddCommand(cmdFileZip)
	cmdFile.AddCommand(cmdFileUnzip)
	cmdFile.AddCommand(cmdFileRm)
	cmdFile.AddCommand(cmdFileTouch)
	cmdFile.AddCommand(cmdFileMkdir)
}
