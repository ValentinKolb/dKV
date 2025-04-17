package common

import (
	"fmt"
	"github.com/lni/dragonboat/v4/config"
	"math"
	"sort"
	"strconv"
	"strings"
)

// --------------------------------------------------------------------------
// helper functions for to interface with Dragonboat (for the server util)
// --------------------------------------------------------------------------

// Dragonboat uses RTT (Round Trip Time) to determine the timing of elections and heartbeats.
// These default values are selected according to the RAFT Paper
const (
	electionRTTFactor  = 10
	heartbeatRTTFactor = 1
)

// ToDragonboatConfig converts the ServerConfig to Dragonboat Config
func (c *ServerConfig) ToDragonboatConfig(shardId uint64) config.Config {
	return config.Config{
		ReplicaID:          c.ReplicaID,
		ShardID:            shardId,
		ElectionRTT:        electionRTTFactor,  // = c.RTTMillisecond * 10
		HeartbeatRTT:       heartbeatRTTFactor, // = c.RTTMillisecond * 2
		CheckQuorum:        true,
		SnapshotEntries:    c.SnapshotEntries,
		CompactionOverhead: c.CompactionOverhead,
		MaxInMemLogSize:    0,
	}
}

// ToNodeHostConfig creates a NodeHostConfig for Dragonboat
func (c *ServerConfig) ToNodeHostConfig() config.NodeHostConfig {
	return config.NodeHostConfig{
		WALDir:         c.DataDir,
		NodeHostDir:    c.DataDir,
		RTTMillisecond: c.RTTMillisecond,
		RaftAddress:    c.ClusterMembers[c.ReplicaID],
	}
}

// --------------------------------------------------------------------------
// RPC server configuration struct
// --------------------------------------------------------------------------

type ServerShardType string

const (
	ShardTypeLocalIStore        ServerShardType = "local store"
	ShardTypeRemoteIStore                       = "remote store"
	ShardTypeLocalILockManager                  = "local lock manager"
	ShardTypeRemoteILockManager                 = "remote lock manager"
)

type ServerShard struct {
	// ShardID is the ID of the shard
	ShardID uint64
	// Store is the store for the shard
	Type ServerShardType
}

// ServerConfig holds all configuration parameters for the RAFT cluster.
type ServerConfig struct {
	// whether to start the server in single node mode or in a cluster
	Shards []ServerShard

	// Dragenboat parameters
	RTTMillisecond     uint64
	SnapshotEntries    uint64
	CompactionOverhead uint64
	DataDir            string
	ReplicaID          uint64
	ClusterMembers     map[uint64]string

	// remote kvStore parameters
	TimeoutSecond int64

	// HTTP api settings
	Endpoint string

	// Logging configuration
	LogLevel string
}

// HasRemoteShard checks if the configuration contains any remote shards
func (c *ServerConfig) HasRemoteShard() bool {
	for _, shard := range c.Shards {
		if shard.Type == ShardTypeRemoteIStore || shard.Type == ShardTypeRemoteILockManager {
			return true
		}
	}
	return false
}

// String returns a formatted string representation of the configuration
func (c *ServerConfig) String() string {
	var sb strings.Builder

	// Create helper functions for consistent formatting
	addSection := func(title string) {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s\n", strings.ToUpper(title)))
	}

	addField := func(name, value string) {
		sb.WriteString(fmt.Sprintf("  %-22s: %s\n", name, value))
	}

	// RPC settings
	addSection("RPC Server")
	addField("Endpoint", c.Endpoint)
	addField("Timeout", fmt.Sprintf("%d sec", c.TimeoutSecond))

	// Logging configuration
	addSection("Logging")
	addField("Log Level", c.LogLevel)

	// Shards
	addSection("Shards")
	for _, shard := range c.Shards {
		addField(strconv.FormatUint(shard.ShardID, 10), string(shard.Type))
	}

	if c.HasRemoteShard() {
		// Node Identity
		addSection("Node Identity")
		addField("RAFT Address", c.ClusterMembers[c.ReplicaID])
		addField("Node ID", strconv.FormatUint(c.ReplicaID, 10))

		// RAFT parameters
		addSection("RAFT Parameters")
		addField("Round Trip Time (ms)", fmt.Sprintf("%d ms", c.RTTMillisecond))
		addField("Election RTT (ms)", fmt.Sprintf("%d", c.RTTMillisecond*electionRTTFactor))
		addField("Heartbeat RTT (ms)", fmt.Sprintf("%d", c.RTTMillisecond*heartbeatRTTFactor))
		addField("Check Quorum", fmt.Sprintf("%t", true))
		addField("Snapshot Entries", fmt.Sprintf("%d", c.SnapshotEntries))
		addField("Compaction Overhead", fmt.Sprintf("%d", c.CompactionOverhead))
		addField("Timeout", fmt.Sprintf("%d sec", c.TimeoutSecond))

		// Storage
		addSection("Storage")
		addField("Data Directory", c.DataDir)

		// ConfServerModeMultiNode configuration
		addSection("ConfServerModeMultiNode")
		sb.WriteString("  Initial ConfServerModeMultiNode Members:\n")

		// Sort keys for consistent output
		var keys []uint64
		for k := range c.ClusterMembers {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("    Node %d: %s\n", k, c.ClusterMembers[k]))
		}
	}
	return sb.String()
}

// --------------------------------------------------------------------------
// RPC client configuration struct
// --------------------------------------------------------------------------

type ClientConfig struct {
	Endpoints              []string
	TimeoutSecond          int
	RetryCount             int
	ConnectionsPerEndpoint int
}

// String returns a formatted string representation of the client configuration
func (c *ClientConfig) String() string {
	var sb strings.Builder

	// Create helper functions for consistent formatting
	addSection := func(title string) {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s\n", strings.ToUpper(title)))
	}

	addField := func(name, value string) {
		sb.WriteString(fmt.Sprintf("  %-22s: %s\n", name, value))
	}

	// General Client Settings
	addSection("Client Configuration")
	addField("Timeout", fmt.Sprintf("%d sec", c.TimeoutSecond))
	addField("Retry Count", strconv.Itoa(c.RetryCount))
	addField("Connections Per Endpoint", strconv.Itoa(int(math.Max(1, float64(c.ConnectionsPerEndpoint)))))

	// Endpoints
	addSection("Endpoints")
	for i, endpoint := range c.Endpoints {
		addField(strconv.Itoa(i), endpoint)
	}

	return sb.String()
}
