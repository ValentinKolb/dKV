package maple

import (
	"github.com/ValentinKolb/dKV/lib/db"
	dbtesting "github.com/ValentinKolb/dKV/lib/db/testing"
	"testing"
)

func Test(t *testing.T) {
	dbtesting.RunKVDBTests(t, "MapleDB", func() db.KVDB {
		return NewMapleDB(nil)
	})
}

func Benchmark(t *testing.B) {
	dbtesting.RunKVDBBenchmarks(t, "MapleDB", func() db.KVDB {
		return NewMapleDB(nil)
	})
}

/*
BENCH RESULTS (Apple M1 Max, 64GB RAM, macOS 15.3.2, go version go1.24.1 darwin/arm64):

goos: darwin
goarch: arm64
pkg: github.com/ValentinKolb/dKV/lib/db/engines/maple
cpu: Apple M1 Max
Benchmark
Benchmark/Set
Benchmark/Set-10         	 5931741	       172.7 ns/op
Benchmark/SetExisting
Benchmark/SetExisting-10 	 8521070	       132.8 ns/op
Benchmark/SetLargeValue
Benchmark/SetLargeValue-10         	   50612	     32671 ns/op
Benchmark/SetWithExpiry
Benchmark/SetWithExpiry-10         	 5485984	       245.4 ns/op
Benchmark/Get
Benchmark/Get-10                   	11817172	        97.97 ns/op
Benchmark/GetWithExpiry
Benchmark/GetWithExpiry-10         	10115143	       102.6 ns/op
Benchmark/Delete
Benchmark/Delete-10                	13985959	        76.90 ns/op
Benchmark/Has
Benchmark/Has-10                   	14844438	       102.5 ns/op
Benchmark/Has(not)
Benchmark/Has(not)-10              	22889605	        73.14 ns/op
Benchmark/SaveLoad
Benchmark/SaveLoad/Save
Benchmark/SaveLoad/Save-10         	     602	   1975853 ns/op
Benchmark/SaveLoad/Load
Benchmark/SaveLoad/Load-10         	       6	 176277167 ns/op
Benchmark/MixedUsage
Benchmark/MixedUsage-10            	12662244	        91.92 ns/op
Benchmark/MixedUsageWithExpiry
Benchmark/MixedUsageWithExpiry-10  	 9093520	       112.8 ns/op
PASS

Process finished with the exit code 0
*/
