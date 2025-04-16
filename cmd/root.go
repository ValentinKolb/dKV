package cmd

import (
	"github.com/ValentinKolb/dKV/cmd/kv"
	"github.com/ValentinKolb/dKV/cmd/lock"
	"github.com/ValentinKolb/dKV/cmd/serve"
	"github.com/ValentinKolb/dKV/cmd/util"
	"github.com/spf13/cobra"
	"os"
)

var (

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "dkv",
		Short: "dKV is a dstore key-value store",
		Long: `dKV is a dstore key-value store built on Dragonboat RAFT.
This application can run as a server or client for interacting with the dKV store.`,
	}
)

func init() {
	// Add Commands
	RootCmd.AddCommand(serve.ServeCmd)
	RootCmd.AddCommand(kv.KeyValueCommands)
	RootCmd.AddCommand(lock.LockCommands)

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
