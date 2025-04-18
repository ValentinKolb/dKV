package util

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/serializer"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/ValentinKolb/dKV/rpc/transport/http"
	"github.com/ValentinKolb/dKV/rpc/transport/tcp"
	"github.com/ValentinKolb/dKV/rpc/transport/unix"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

const (
	// Wrap is the number of characters to Wrap the help text at
	Wrap int = 50
)

// WrapString wraps a string at Wrap characters
func WrapString(text string) string {
	var wrappedLines []string
	var currentLine strings.Builder
	lineWidth := 0

	for _, word := range strings.Fields(text) {
		wordWidth := len(word)

		// Check if we need to wrap
		if lineWidth > 0 && lineWidth+1+wordWidth > Wrap {
			wrappedLines = append(wrappedLines, currentLine.String())
			currentLine.Reset()
			lineWidth = 0
		}

		// Add space before word (if not first word on line)
		if lineWidth > 0 {
			currentLine.WriteString(" ")
			lineWidth++
		}

		// Add the word
		currentLine.WriteString(word)
		lineWidth += wordWidth
	}

	// Add any remaining text
	if currentLine.Len() > 0 {
		wrappedLines = append(wrappedLines, currentLine.String())
	}

	return strings.Join(wrappedLines, "\n")
}

// SetupRPCClientFlags adds common RPC connection flags to a command
func SetupRPCClientFlags(cmd *cobra.Command) {
	key := "timeout"
	cmd.PersistentFlags().Int(key, 10, WrapString("The timeout in seconds of the client"))

	key = "transport-endpoints"
	cmd.PersistentFlags().String(key, "http://localhost:8080", WrapString("The address of the dKV server. For transports that support load balancing, multiple endpoints can be specified as a comma-separated list"))

	key = "transport-conn-per-endpoint"
	cmd.PersistentFlags().Int(key, 1, WrapString("Simultaneous connections per endpoint - for transports that support this feature"))

	key = "transport-retries"
	cmd.PersistentFlags().Int(key, 3, WrapString("How many times to retry the request"))

	key = "transport-write-buffer"
	cmd.PersistentFlags().Int(key, 512, WrapString("The size of the write buffer for the transport (in KB, ignored for http)"))

	key = "transport-read-buffer"
	cmd.PersistentFlags().Int(key, 512, WrapString("The size of the read buffer for the transport (in KB, ignored for http)"))

	key = "transport-tcp-nodelay"
	cmd.PersistentFlags().Bool(key, true, WrapString("Whether to enable TCP_NODELAY for the transport (only for TCPConf)"))

	key = "transport-tcp-keepalive"
	cmd.PersistentFlags().Int(key, 0, WrapString("The keepalive interval for the transport (in seconds, only for TCPConf)"))

	key = "transport-tcp-linger"
	cmd.PersistentFlags().Int(key, 0, WrapString("The linger time for the transport (in seconds, only for TCPConf)"))
}

// InitClientConfig initializes configuration from environment variables
func InitClientConfig() {
	// load env files
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")

	// initialize viper
	viper.SetEnvPrefix("dkv")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match
}

// GetClientConfig reads client configuration from viper
func GetClientConfig() *common.ClientConfig {
	conf := &common.ClientConfig{
		TimeoutSecond: viper.GetInt("timeout"),
		Transport: common.ClientTransportConfig{
			RetryCount:             viper.GetInt("transport-retries"),
			Endpoints:              strings.Split(viper.GetString("transport-endpoints"), ","),
			ConnectionsPerEndpoint: viper.GetInt("transport-conn-per-endpoint"),
			SocketConf: common.SocketConf{
				WriteBufferSize: viper.GetInt("transport-write-buffer") * 1024,
				ReadBufferSize:  viper.GetInt("transport-read-buffer") * 1024,
			},
			TCPConf: common.TCPConf{
				TCPKeepAliveSec: viper.GetInt("transport-tcp-keepalive"),
				TCPLingerSec:    viper.GetInt("transport-tcp-linger"),
				TCPNoDelay:      viper.GetBool("transport-tcp-nodelay"),
			},
		},
	}

	return conf
}

// GetSerializer creates a serializer based on configuration
func GetSerializer() (serializer.IRPCSerializer, error) {
	switch viper.GetString("serializer") {
	case "json":
		return serializer.NewJSONSerializer(), nil
	case "gob":
		return serializer.NewGOBSerializer(), nil
	case "binary":
		return serializer.NewBinarySerializer(), nil
	default:
		return nil, fmt.Errorf("invalid serializer %s", viper.GetString("serializer"))
	}
}

// GetTransport creates transport based on configuration
func GetTransport() (transport.IRPCClientTransport, error) {
	switch viper.GetString("transport") {
	case "http":
		return http.NewHttpClientTransport(), nil
	case "tcp":
		return tcp.NewTCPClientTransport(), nil
	case "unix":
		return unix.NewUnixClientTransport(), nil
	default:
		return nil, fmt.Errorf("invalid transport %s", viper.GetString("transport"))
	}
}

// GetShardID retrieves the configured shard ID
func GetShardID() uint64 {
	return uint64(viper.GetInt("shard"))
}

// BindCommandFlags binds a command's flags to viper
func BindCommandFlags(cmd *cobra.Command) error {
	return viper.BindPFlags(cmd.Flags())
}
