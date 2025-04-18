package common

import (
	"fmt"
	"github.com/lni/dragonboat/v4/config"
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

	// Logging configuration
	LogLevel string

	// Transport configuration
	Transport ServerTransportConfig
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

	// RPC settings
	addSection("RPC Server", &sb)
	addField("Timeout", fmt.Sprintf("%d sec", c.TimeoutSecond), &sb)

	// Transport settings
	sb.WriteString(c.Transport.String())

	// Logging configuration
	addSection("Logging", &sb)
	addField("Log Level", c.LogLevel, &sb)

	// Shards
	addSection("Shards", &sb)
	for _, shard := range c.Shards {
		addField(strconv.FormatUint(shard.ShardID, 10), string(shard.Type), &sb)
	}

	if c.HasRemoteShard() {
		// Node Identity
		addSection("Node Identity", &sb)
		addField("RAFT Address", c.ClusterMembers[c.ReplicaID], &sb)
		addField("Node ID", strconv.FormatUint(c.ReplicaID, 10), &sb)

		// RAFT parameters
		addSection("RAFT Parameters", &sb)
		addField("Round Trip Time (ms)", fmt.Sprintf("%d ms", c.RTTMillisecond), &sb)
		addField("Election RTT (ms)", fmt.Sprintf("%d", c.RTTMillisecond*electionRTTFactor), &sb)
		addField("Heartbeat RTT (ms)", fmt.Sprintf("%d", c.RTTMillisecond*heartbeatRTTFactor), &sb)
		addField("Check Quorum", fmt.Sprintf("%t", true), &sb)
		addField("Snapshot Entries", fmt.Sprintf("%d", c.SnapshotEntries), &sb)
		addField("Compaction Overhead", fmt.Sprintf("%d", c.CompactionOverhead), &sb)
		addField("Timeout", fmt.Sprintf("%d sec", c.TimeoutSecond), &sb)

		// Storage
		addSection("Storage", &sb)
		addField("Data Directory", c.DataDir, &sb)

		// ConfServerModeMultiNode configuration
		addSection("ConfServerModeMultiNode", &sb)
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
	Transport     ClientTransportConfig
	TimeoutSecond int
}

// String returns a formatted string representation of the client configuration
func (c *ClientConfig) String() string {
	var sb strings.Builder

	// General Client Settings
	addSection("Client Configuration", &sb)
	addField("Timeout", fmt.Sprintf("%d sec", c.TimeoutSecond), &sb)
	// Transport settings
	sb.WriteString(c.Transport.String())

	return sb.String()
}

// ---------------------------------------------------------------
// Transport specific configuration
// ---------------------------------------------------------------

type TCPConf struct {
	TCPNoDelay      bool
	TCPKeepAliveSec int
	TCPLingerSec    int
}

func (c *TCPConf) String() string {
	var sb strings.Builder

	addField("TCPConf No Delay", strconv.FormatBool(c.TCPNoDelay), &sb)
	addField("TCPConf Keep Alive (sec)", strconv.Itoa(c.TCPKeepAliveSec), &sb)
	addField("TCPConf Linger (sec)", strconv.Itoa(c.TCPLingerSec), &sb)

	return sb.String()
}

type SocketConf struct {
	WriteBufferSize int
	ReadBufferSize  int
}

func (c *SocketConf) String() string {
	var sb strings.Builder

	addField("Write Buffer Size", strconv.Itoa(c.WriteBufferSize), &sb)
	addField("Read Buffer Size", strconv.Itoa(c.ReadBufferSize), &sb)

	return sb.String()
}

type ServerTransportConfig struct {
	TCPConf
	SocketConf
	// Server specific settings
	WorkersPerConn int
	Endpoint       string
}

func (c *ServerTransportConfig) String() string {
	var sb strings.Builder

	addField("Workers Per Connection", strconv.Itoa(c.WorkersPerConn), &sb)
	addField("Endpoint", c.Endpoint, &sb)
	sb.WriteString(c.SocketConf.String())
	sb.WriteString(c.TCPConf.String())

	return sb.String()
}

type ClientTransportConfig struct {
	TCPConf
	SocketConf
	// Client specific settings
	ConnectionsPerEndpoint int
	Endpoints              []string
	RetryCount             int
}

func (c *ClientTransportConfig) String() string {
	var sb strings.Builder

	// Endpoints
	addSection("Endpoints", &sb)
	for i, endpoint := range c.Endpoints {
		addField("Server "+strconv.Itoa(i+1), endpoint, &sb)
	}

	addSection("Configuration", &sb)

	addField("Connections Per Endpoint", strconv.Itoa(c.ConnectionsPerEndpoint), &sb)

	addField("Retry Count", strconv.Itoa(c.RetryCount), &sb)

	sb.WriteString(c.SocketConf.String())
	sb.WriteString(c.TCPConf.String())

	return sb.String()
}

// -------------------------------------------------------------
// Helper functions
// -------------------------------------------------------------

func addSection(title string, sb *strings.Builder) {
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s\n", strings.ToUpper(title)))
}

func addField(name, value string, sb *strings.Builder) {
	sb.WriteString(fmt.Sprintf("  %-22s: %s\n", name, value))
}
