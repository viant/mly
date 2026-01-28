package router

import (
	"fmt"
	"strconv"
	"testing"
)

// Benchmark different approaches to generating batch keys
// These benchmarks compare key generation strategies for the ForceBatchSize1 path

var (
	sampleModelName = "model_name_example"
	sampleOffset    = 12345
	result          string // prevent compiler optimization
)

func BenchmarkBatchKey_Sprintf(b *testing.B) {
	var r string
	for i := 0; i < b.N; i++ {
		r = fmt.Sprintf("%s#%d", sampleModelName, sampleOffset)
	}
	result = r
}

func BenchmarkBatchKey_StrconcatStrconv(b *testing.B) {
	var r string
	for i := 0; i < b.N; i++ {
		r = sampleModelName + "#" + strconv.Itoa(sampleOffset)
	}
	result = r
}

func BenchmarkBatchKey_StrconcatFormatInt(b *testing.B) {
	var r string
	for i := 0; i < b.N; i++ {
		r = sampleModelName + "#" + strconv.FormatInt(int64(sampleOffset), 10)
	}
	result = r
}

// Benchmark with varying model name lengths
func BenchmarkBatchKey_ShortName_Sprintf(b *testing.B) {
	name := "m1"
	var r string
	for i := 0; i < b.N; i++ {
		r = fmt.Sprintf("%s#%d", name, i%1000)
	}
	result = r
}

func BenchmarkBatchKey_ShortName_Strconcat(b *testing.B) {
	name := "m1"
	var r string
	for i := 0; i < b.N; i++ {
		r = name + "#" + strconv.Itoa(i%1000)
	}
	result = r
}

func BenchmarkBatchKey_LongName_Sprintf(b *testing.B) {
	name := "very_long_model_name_with_many_characters_for_testing"
	var r string
	for i := 0; i < b.N; i++ {
		r = fmt.Sprintf("%s#%d", name, i%1000)
	}
	result = r
}

func BenchmarkBatchKey_LongName_Strconcat(b *testing.B) {
	name := "very_long_model_name_with_many_characters_for_testing"
	var r string
	for i := 0; i < b.N; i++ {
		r = name + "#" + strconv.Itoa(i%1000)
	}
	result = r
}

// Benchmark the map lookup with generated keys (more realistic scenario)
func BenchmarkBatchKey_MapLookup_Sprintf(b *testing.B) {
	m := make(map[string]int)
	names := []string{"model1", "model2", "model3", "model4", "model5"}

	// Pre-populate map
	for _, name := range names {
		for j := 0; j < 100; j++ {
			m[fmt.Sprintf("%s#%d", name, j)] = j
		}
	}

	b.ResetTimer()
	var sum int
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%s#%d", names[i%5], i%100)
		sum += m[key]
	}
	_ = sum
}

func BenchmarkBatchKey_MapLookup_Strconcat(b *testing.B) {
	m := make(map[string]int)
	names := []string{"model1", "model2", "model3", "model4", "model5"}

	// Pre-populate map
	for _, name := range names {
		for j := 0; j < 100; j++ {
			m[name+"#"+strconv.Itoa(j)] = j
		}
	}

	b.ResetTimer()
	var sum int
	for i := 0; i < b.N; i++ {
		key := names[i%5] + "#" + strconv.Itoa(i%100)
		sum += m[key]
	}
	_ = sum
}

// Benchmark allocations
func BenchmarkBatchKey_Allocs_Sprintf(b *testing.B) {
	b.ReportAllocs()
	var r string
	for i := 0; i < b.N; i++ {
		r = fmt.Sprintf("%s#%d", sampleModelName, i%1000)
	}
	result = r
}

func BenchmarkBatchKey_Allocs_Strconcat(b *testing.B) {
	b.ReportAllocs()
	var r string
	for i := 0; i < b.N; i++ {
		r = sampleModelName + "#" + strconv.Itoa(i%1000)
	}
	result = r
}
