package kv

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"github.com/ValentinKolb/dKV/cmd/util"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	perfTestCmd = &cobra.Command{
		Use:     "perf",
		Short:   "Performance testing tool for dKV servers",
		Long:    "",
		RunE:    run,
		PreRunE: processPerfConfig,
	}
	perfKeyPrefix        = "__test"
	perfValueSizeBytes   = -1
	perfLargeValueSizeKB = -1
	perfNumThreads       = -1
	perfKeySpread        = -1
	perfSkip             = make([]string, 0)

	integrationTest = &cobra.Command{
		Use:     "test",
		Short:   "Run Integration tests on the server",
		Long:    "",
		RunE:    runTests,
		PreRunE: processPerfConfig,
	}
)

func init() {
	// add flags
	key := "skip"
	perfTestCmd.PersistentFlags().String(key, "", util.WrapString("Benchmarks to skip (comma separated - e.g. set,get)"))
	key = "threads"
	perfTestCmd.PersistentFlags().Int(key, 10, util.WrapString("Number of threads to use for the benchmark"))
	integrationTest.PersistentFlags().Int(key, 10, util.WrapString("Number of threads to use for the benchmark"))
	key = "value-size"
	perfTestCmd.PersistentFlags().Int(key, 128, util.WrapString("The size of the value used for testing (in Bytes)"))
	key = "large-value-size"
	perfTestCmd.PersistentFlags().Int(key, 512, util.WrapString("How large the value for the set-large test should be (in Kilo Bytes)"))
	key = "keys"
	perfTestCmd.PersistentFlags().Int(key, 100, util.WrapString("How many different keys to use for the tests"))
	key = "csv"
	perfTestCmd.Flags().String(key, "", util.WrapString("Optional path to save benchmark results as CSV"))
}

func processPerfConfig(cmd *cobra.Command, _ []string) error {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// Read the configuration from the command line flags and environment variables
	perfValueSizeBytes = viper.GetInt("value-size")
	perfLargeValueSizeKB = viper.GetInt("large-value-size")
	perfKeySpread = viper.GetInt("keys")
	perfNumThreads = viper.GetInt("threads")
	perfSkip = strings.Split(viper.GetString("skip"), ",")

	return nil
}

func run(_ *cobra.Command, _ []string) error {

	fmt.Println("Benchmarking tool for dKV servers")

	// Print configuration
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println(util.GetClientConfig().String())
	fmt.Printf("Threads: %d\n", perfNumThreads)
	fmt.Println()

	fmt.Println("staring benchmarks...")

	// Create results map
	results := make(map[string]testing.BenchmarkResult)

	setResult := testing.Benchmark(func(b *testing.B) {
		if shouldSkip("set") {
			return
		}

		// prepare keys
		getKey, iter := getKeys("set")
		value := make([]byte, perfValueSizeBytes)

		// cleanup
		b.Cleanup(func() {
			iter(func(k string) {
				err := rpcStore.Delete(k)
				if err != nil {
					log.Printf("(set) - error deleting key: %v\n", err)
				}
			})
		})

		b.SetParallelism(perfNumThreads)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				err := rpcStore.Set(getKey(counter), value)
				if err != nil {
					log.Printf("(set) - error setting key: %v\n", err)
				}
			}
		})
	})

	results["set"] = setResult
	printResult("set", setResult)

	setLargeValueResult := testing.Benchmark(func(b *testing.B) {
		if shouldSkip("set-large") {
			return
		}

		// prepare large value
		largeValue := make([]byte, perfLargeValueSizeKB*1024)

		// prepare keys
		getKey, iter := getKeys("set-large")

		// cleanup
		b.Cleanup(func() {
			iter(func(k string) {
				err := rpcStore.Delete(k)
				if err != nil {
					log.Printf("(set-large) - error deleting key: %v\n", err)
				}
			})
		})

		b.SetParallelism(perfNumThreads)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter := 0
				for pb.Next() {
					err := rpcStore.Set(getKey(counter), largeValue)
					if err != nil {
						log.Printf("(set-large) - error setting key: %v", err)
					}
				}
			}
		})

	})

	results["set-large"] = setLargeValueResult
	printResult("large-set", setLargeValueResult)

	getResult := testing.Benchmark(func(b *testing.B) {
		if shouldSkip("get") {
			return
		}

		// prepare keys
		getKey, iter := getKeys("get")
		value := make([]byte, perfValueSizeBytes)

		// set keys
		iter(func(k string) {
			err := rpcStore.Set(k, value)
			if err != nil {
				log.Printf("(get) - error setting key: %v\n", err)
			}
		})

		// cleanup
		b.Cleanup(func() {
			iter(func(k string) {
				err := rpcStore.Delete(k)
				if err != nil {
					log.Printf("(get) - error deleting key: %v\n", err)
				}
			})
		})

		b.SetParallelism(perfNumThreads)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				_, _, err := rpcStore.Get(getKey(counter))
				if err != nil {
					log.Printf("(get) - error getting key: %v\n", err)
				}
				counter++
			}
		})
	})

	results["get"] = getResult
	printResult("get", getResult)

	deleteResult := testing.Benchmark(func(b *testing.B) {
		if shouldSkip("delete") {
			return
		}

		// prepare keys
		getKey, iter := getKeys("delete")
		value := make([]byte, perfValueSizeBytes)

		// set keys
		iter(func(k string) {
			err := rpcStore.Set(k, value)
			if err != nil {
				log.Printf("(delete) - error setting key: %v\n", err)
			}
		})

		// cleanup
		b.Cleanup(func() {
			iter(func(k string) {
				err := rpcStore.Delete(k)
				if err != nil {
					log.Printf("(delete) - error deleting key: %v\n", err)
				}
			})
		})

		b.SetParallelism(perfNumThreads)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				err := rpcStore.Delete(getKey(counter))
				if err != nil {
					log.Printf("(delete) - error deleting key: %v\n", err)
				}
				counter++
			}
		})
	})

	results["delete"] = deleteResult
	printResult("delete", deleteResult)

	hasResult := testing.Benchmark(func(b *testing.B) {
		if shouldSkip("has") {
			return
		}

		// prepare keys
		getKey, iter := getKeys("has")
		value := make([]byte, perfValueSizeBytes)

		// set keys
		iter(func(k string) {
			err := rpcStore.Set(k, value)
			if err != nil {
				log.Printf("(has) - error setting key: %v\n", err)
			}
		})

		// cleanup
		b.Cleanup(func() {
			iter(func(k string) {
				err := rpcStore.Delete(k)
				if err != nil {
					log.Printf("(has) - error deleting key: %v\n", err)
				}
			})
		})

		b.SetParallelism(perfNumThreads)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				_, err := rpcStore.Has(getKey(counter))
				if err != nil {
					log.Printf("(has) - error checking key: %v\n", err)
				}
				counter++
			}
		})
	})

	results["has"] = hasResult
	printResult("has", hasResult)

	hasNotResult := testing.Benchmark(func(b *testing.B) {
		if shouldSkip("has-not") {
			return
		}

		b.SetParallelism(perfNumThreads)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				key := fmt.Sprintf("%s/has-not-%d", perfKeyPrefix, counter%100)
				_, err := rpcStore.Has(key) // ignore error (404 expected)
				if err != nil {
					log.Printf("(has-not) - error checking key: %v\n", err)
				}
				counter++
			}
		})
	})

	results["has-not"] = hasNotResult
	printResult("has-not", hasNotResult)

	mixedUsageResult := testing.Benchmark(func(b *testing.B) {
		if shouldSkip("mixed") {
			return
		}

		// prepare keys
		getKey, iter := getKeys("mixed")
		value := make([]byte, perfValueSizeBytes)

		// set keys
		iter(func(k string) {
			err := rpcStore.Set(k, value)
			if err != nil {
				log.Printf("(mixed) - error setting key: %v\n", err)
			}
		})

		// cleanup
		b.Cleanup(func() {
			iter(func(k string) {
				err := rpcStore.Delete(k)
				if err != nil {
					log.Printf("(mixed) - error deleting key: %v\n", err)
				}
			})
		})

		b.SetParallelism(perfNumThreads)

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				key := getKey(counter)
				var err error
				switch counter % 4 {
				case 0: // set
					err = rpcStore.Set(key, value)
				case 1: // get
					_, _, err = rpcStore.Get(key)
				case 2: // delete
					err = rpcStore.Delete(key)
				case 3: // has
					_, err = rpcStore.Has(key)
				}

				if err != nil {
					log.Printf("(mixed) - error performing operation (%s): %v\n", counter%4, err)
				}
				counter++
			}
		})
	})

	results["mixed"] = mixedUsageResult
	printResult("mixed", mixedUsageResult)

	// Write results to csv is specified
	if csvPath := viper.GetString("csv"); csvPath != "" {
		fmt.Printf("\nExporting results to CSV: %s\n", csvPath)
		if err := writeResultsToCSV(csvPath, results, util.GetClientConfig()); err != nil {
			return fmt.Errorf("failed to export results to CSV: %v", err)
		}
		fmt.Println("Export complete")
	}

	return nil
}

// --------------------------------------------------------------------------
// Helper
// --------------------------------------------------------------------------

func shouldSkip(test string) bool {
	// Check if the test is in the skip list
	for _, skip := range perfSkip {
		if test == skip {
			return true
		}
	}
	return false
}

// creates an array of test keys and functions to work with them
func getKeys(prefix string) (func(int) string, func(func(string))) {
	keys := make([]string, perfKeySpread)
	for i := 0; i < perfKeySpread; i++ {
		keys[i] = fmt.Sprintf("%s-%s-%d", perfKeyPrefix, prefix, i)
	}

	// Function to get a key by index (with wraparound)
	getKey := func(i int) string {
		return keys[i%perfKeySpread]
	}

	// Function to iterate over all keys and apply a function to each
	iterateKeys := func(fn func(string)) {
		for _, key := range keys {
			fn(key)
		}
	}

	return getKey, iterateKeys
}

// printResult prints the result of a benchmark test in a formatted way
func printResult(test string, result testing.BenchmarkResult) {
	if result.NsPerOp() == 0 {
		fmt.Printf("%-20sskipped\n", test)
		return
	}

	nsPerOp := math.Max(float64(result.NsPerOp()), 1) // prevent division by zero
	opsPerSec := 1.0 / (nsPerOp / 1e9)

	// Print the formatted result
	fmt.Printf("%-20s%.0fns/op (%s/op)\t%.0f ops/sec\n", test, nsPerOp, time.Duration(nsPerOp), opsPerSec)
}

// writeResultsToCSV writes benchmark results to a CSV file
func writeResultsToCSV(csvPath string, results map[string]testing.BenchmarkResult, config *common.ClientConfig) error {
	file, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Test", "NsPerOp", "DurationPerOp", "OpsPerSec", "Skipped",
		"Endpoints", "TimeoutSec", "RetryCount", "ConnectionsPerEndpoint",
		"ShardID", "Serializer", "Transport",
		"Threads", "LargeValueSizeKB", "Keys Count",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %v", err)
	}

	// Write test results
	for test, result := range results {
		var nsPerOp float64
		var opsPerSec float64
		var skipped string

		if result.NsPerOp() == 0 {
			skipped = "true"
			nsPerOp = 0
			opsPerSec = 0
		} else {
			skipped = "false"
			nsPerOp = math.Max(float64(result.NsPerOp()), 1)
			opsPerSec = 1.0 / (nsPerOp / 1e9)
		}

		row := []string{
			test,
			fmt.Sprintf("%.0f", nsPerOp),
			time.Duration(nsPerOp).String(),
			fmt.Sprintf("%.0f", opsPerSec),
			skipped,
			strings.Join(config.Transport.Endpoints, ";"),
			strconv.Itoa(config.TimeoutSecond),
			strconv.Itoa(config.Transport.RetryCount),
			strconv.Itoa(config.Transport.ConnectionsPerEndpoint),
			strconv.FormatUint(util.GetShardID(), 10),
			viper.GetString("serializer"),
			viper.GetString("transport"),
			strconv.Itoa(perfNumThreads),
			strconv.Itoa(perfLargeValueSizeKB),
			strconv.Itoa(perfKeySpread),
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row for test %s: %v", test, err)
		}
	}

	return nil
}

// --------------------------------------------------------------------------
// Integration tests
// --------------------------------------------------------------------------

func runTests(_ *cobra.Command, _ []string) error {

	fmt.Println("Integration testing tool for dKV servers")

	// Print configuration
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println(util.GetClientConfig().String())
	fmt.Printf("Threads: %d\n", perfNumThreads)
	fmt.Println()

	fmt.Println("staring tests...")

	runTest := func() error {
		value := make([]byte, 8)
		_, err := rand.Read(value)
		if err != nil {
			return fmt.Errorf("(integration-test) - error generating random bytes: %v\n", err)
		}

		key := fmt.Sprintf("%s-%x", perfKeyPrefix, value)

		// cleanup
		defer rpcStore.Delete(key)

		defer func(rpcStore store.IStore, key string) {
			err := rpcStore.Delete(key)
			if err != nil {
				fmt.Println("Failed to delete key", key, err)
			}
		}(rpcStore, key)

		// First check if key exists
		exists, err := rpcStore.Has(key)

		if err != nil {
			return fmt.Errorf("(integration-test) - error checking key: %v\n", err)
		}

		if exists {
			return fmt.Errorf("(integration-test) - key should not exist befor test: %s", key)

		}

		// Set the key
		err = rpcStore.Set(key, value)

		if err != nil {
			return fmt.Errorf("(integration-test) - error setting key: %v\n", err)

		}

		// Get the key
		gotValue, ok, err := rpcStore.Get(key)

		if err != nil {
			return fmt.Errorf("(integration-test) - error getting key: %v\n", err)

		}

		if !ok {
			return fmt.Errorf("(integration-test) - key should exist after set: %s", key)
		}

		// Check if the value is correct
		if !bytes.Equal(gotValue, value) {
			return fmt.Errorf("(integration-test) - value mismatch for key %s: expected %x, got %x", key, value, gotValue)

		}

		// Expire the key
		err = rpcStore.Expire(key)

		if err != nil {
			return fmt.Errorf("(integration-test) - error expiring key: %v\n", err)

		}

		// Check if the value is still there
		_, ok, err = rpcStore.Get(key)

		if err != nil {
			return fmt.Errorf("(integration-test) - error getting key after expire: %v\n", err)

		}

		if ok {
			return fmt.Errorf("(integration-test) - value should not exist after expire: %s", key)

		}

		// Check if has works
		if exists, err := rpcStore.Has(key); err != nil {
			return fmt.Errorf("(integration-test) - error checking key after expire: %v\n", err)

		} else if !exists {
			return fmt.Errorf("(integration-test) - key should exist after expire: %s", key)

		}

		// Delete the key
		err = rpcStore.Delete(key)

		if err != nil {
			return fmt.Errorf("(integration-test) - error deleting key: %v\n", err)

		}

		// Check if the key is deleted
		exists, err = rpcStore.Has(key)

		if err != nil {
			return fmt.Errorf("(integration-test) - error checking key after delete: %v\n", err)

		}

		return nil
	}

	var errCount uint64 = 0
	wg := sync.WaitGroup{}

	for i := 0; i < perfNumThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				err := runTest()
				if err != nil {
					log.Printf("%v", err)
					atomic.AddUint64(&errCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	fmt.Printf("\nFinished %d test runs with %d errors\n", perfNumThreads*100, errCount)

	return nil
}
