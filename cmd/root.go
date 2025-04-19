package cmd

import (
	"fmt"
	"github.com/ValentinKolb/dKV/cmd/kv"
	"github.com/ValentinKolb/dKV/cmd/lock"
	"github.com/ValentinKolb/dKV/cmd/serve"
	"github.com/ValentinKolb/dKV/cmd/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"runtime"
)

const (
	Version = "1.0.9"
)

var (

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "dkv",
		Short: "distributed key-value store",
		Long: fmt.Sprintf(`dKV (v%s)

A distributed, consistent key-value store library written in Go,
leveraging RAFT consensus for linearizability and fault tolerance.`, Version),
	}
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of dKV",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dKV v%s\n", Version)
		},
	}

	// upgradeCmd represents the upgrade command
	upgradeCmd = &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade dKV to the latest version",
		Long:  `Upgrade dKV to the latest version by downloading and running the installation script.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Upgrading dKV to the latest version...")

			// Get installation path flag
			installPath, _ := cmd.Flags().GetString("path")

			// Get source flag
			fromSource, _ := cmd.Flags().GetBool("source")

			// Prepare command arguments
			scriptURL := "https://raw.githubusercontent.com/ValentinKolb/dKV/refs/heads/main/install.sh"
			var shellCmd *exec.Cmd

			if runtime.GOOS == "windows" {
				fmt.Println("Windows is not supported.")
				os.Exit(1)
			}

			// Base command to download and execute the script
			baseCmd := fmt.Sprintf("curl -s %s | bash", scriptURL)

			// Add options if specified
			options := ""
			if installPath != "" {
				options += fmt.Sprintf(" -- --path=%s", installPath)
			}
			if fromSource {
				if options == "" {
					options = " -- --source"
				} else {
					options += " --source"
				}
			}

			// Combine the command
			cmdStr := baseCmd + options

			// Create and run the command
			shellCmd = exec.Command("bash", "-c", cmdStr)
			shellCmd.Stdout = os.Stdout
			shellCmd.Stderr = os.Stderr

			fmt.Println("Executing:", cmdStr)
			err := shellCmd.Run()
			if err != nil {
				fmt.Printf("Error upgrading dKV: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("dKV has been successfully upgraded!")
		},
	}
)

func init() {
	// Add Commands
	RootCmd.AddCommand(serve.ServeCmd)
	RootCmd.AddCommand(kv.KeyValueCommands)
	RootCmd.AddCommand(lock.LockCommands)
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(upgradeCmd)

	// Add Flags for upgrade command
	upgradeCmd.Flags().String("path", "", "Installation path for the upgraded version")
	upgradeCmd.Flags().Bool("source", false, "Install from source instead of using pre-compiled binaries")

	// Add Flags
	key := "serializer"
	RootCmd.PersistentFlags().String(key, "json", util.WrapString("serializer to use (json, gob, binary)"))
	key = "transport"
	RootCmd.PersistentFlags().String(key, "http", util.WrapString("transport to use (http, tcp, unix)"))
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
