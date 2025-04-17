package kv

import (
	"github.com/ValentinKolb/dKV/cmd/util"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/rpc/client"
	"github.com/spf13/cobra"
)

var (
	rpcStore store.IStore

	// KeyValueCommands represents the KV command group
	KeyValueCommands = &cobra.Command{
		Use:               "kv",
		Short:             "Perform key-value store operations",
		PersistentPreRunE: setupKVClient,
	}
)

func init() {
	// Initialize viper
	cobra.OnInitialize(util.InitClientConfig)

	// Add common RPC flags to the KV command
	util.SetupRPCClientFlags(KeyValueCommands)

	// Set default shard ID for key value operations (different from Lock default)
	KeyValueCommands.PersistentFlags().Int("shard", 100, util.WrapString("ID of the shard to connect to"))

	// Add subcommands
	KeyValueCommands.AddCommand(setCmd)
	KeyValueCommands.AddCommand(setECmd)
	KeyValueCommands.AddCommand(setEIfUnsetCmd)
	KeyValueCommands.AddCommand(getCmd)
	KeyValueCommands.AddCommand(exprCmd)
	KeyValueCommands.AddCommand(delCmd)
	KeyValueCommands.AddCommand(hasCmd)
	KeyValueCommands.AddCommand(perfTestCmd)
}

// setupKVClient initializes the RPC store client
func setupKVClient(cmd *cobra.Command, _ []string) error {
	// Bind command flags to viper
	if err := util.BindCommandFlags(cmd); err != nil {
		return err
	}

	// Get client configuration components
	config := util.GetClientConfig()
	shardId := util.GetShardID()

	// Get serializer and transport
	s, err := util.GetSerializer()
	if err != nil {
		return err
	}

	t, err := util.GetTransport()
	if err != nil {
		return err
	}

	// Create the KV store client
	rpcStore, err = client.NewRPCStore(
		shardId,
		*config,
		t,
		s,
	)

	return err
}
