package lock

import (
	"encoding/hex"
	"fmt"
	"github.com/ValentinKolb/dKV/cmd/util"
	"github.com/ValentinKolb/dKV/lib/lockmgr"
	"github.com/ValentinKolb/dKV/rpc/client"
	"github.com/spf13/cobra"
)

var (
	rpcLockMgr     lockmgr.ILockManager
	acquireTimeout uint64

	// LockCommands represents the lock command group
	LockCommands = &cobra.Command{
		Use:               "lock",
		Short:             "Perform lock operations",
		PersistentPreRunE: setupLockClient,
	}

	// acquireCmd represents the acquire command
	acquireCmd = &cobra.Command{
		Use:   "acquire [key]",
		Short: "Acquire a lock",
		Args:  cobra.ExactArgs(1),
		RunE:  runAcquire,
	}

	// releaseCmd represents the release command
	releaseCmd = &cobra.Command{
		Use:   "release [key] [ownerID]",
		Short: "Release a previously acquired lock",
		Long:  "Release a lock using the key and owner ID. The owner ID is the hex string returned by the acquire command.",
		Args:  cobra.ExactArgs(2),
		RunE:  runRelease,
	}
)

func init() {
	// Initialize viper
	cobra.OnInitialize(util.InitClientConfig)

	// Add subcommands to lock command
	LockCommands.AddCommand(acquireCmd)
	LockCommands.AddCommand(releaseCmd)

	// Add common RPC flags to the lock command
	util.SetupRPCClientFlags(LockCommands)

	// Set default shard ID for lock operations (different from KV default)
	LockCommands.PersistentFlags().Int("shard", 200, util.WrapString("ID of the shard to connect to"))

	// Add flags specific to acquire
	acquireCmd.Flags().Uint64Var(&acquireTimeout, "timeout", 30, "Lock timeout in seconds (0 for no timeout)")
}

// setupLockClient initializes the lock manager client
func setupLockClient(cmd *cobra.Command, _ []string) error {
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

	// Create the lock manager client
	rpcLockMgr, err = client.NewRPCLockMgr(
		shardId,
		*config,
		t,
		s,
	)

	return err
}

// runAcquire handles the acquire lock command
func runAcquire(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Attempt to acquire the lock
	acquired, ownerID, err := rpcLockMgr.AcquireLock(key, acquireTimeout)

	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}

	if !acquired {
		fmt.Printf("aquired=false\n")
		return nil
	}

	// Convert owner ID to hex string for display
	ownerIDHex := hex.EncodeToString(ownerID)
	fmt.Printf("acquired=true, ownedId=%s\n", ownerIDHex)

	return nil
}

// runRelease handles the release lock command
func runRelease(_ *cobra.Command, args []string) error {
	key := args[0]
	ownerIDHex := args[1]

	// Convert hex string owner ID back to bytes
	ownerID, err := hex.DecodeString(ownerIDHex)
	if err != nil {
		return fmt.Errorf("invalid owner ID format: %v", err)
	}

	// Attempt to release the lock
	released, err := rpcLockMgr.ReleaseLock(key, ownerID)

	if err != nil {
		return fmt.Errorf("failed to release lock: %v", err)
	}

	fmt.Printf("released=%v", released)

	return nil
}
