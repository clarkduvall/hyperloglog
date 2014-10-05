package main

import (
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"time"
	"github.com/eclesh/hyperloglog"
)

func hashStr(s string) hash.Hash32 {
	h := fnv.New32()
	h.Write([]byte(s))
	return h
}

func hashStr64(s string) hash.Hash64 {
	h := fnv.New64()
	h.Write([]byte(s))
	return h
}

func main() {
	reg := uint8(14)
	num := 100000
	hll2, _ := hyperloglog.New(1 << reg)
	hll := NewHyperLogLog(reg)
	hllpp := NewHyperLogLogPP(reg)

	start := time.Now()
	for i := 0; i < num; i++ {
		hll2.Add(hashStr(fmt.Sprintf("a", i)).Sum32())
		hll2.Add(hashStr(fmt.Sprintf("a", i)).Sum32())
	}
	elapsed := time.Since(start)
	fmt.Println("Other time elapsed: ", elapsed)
	start = time.Now()
	for i := 0; i < num; i++ {
		hll.Add(hashStr(fmt.Sprintf("a", i)))
		hll.Add(hashStr(fmt.Sprintf("a", i)))
	}
	elapsed = time.Since(start)
	fmt.Println("Mine time elapsed:  ", elapsed)

	start = time.Now()
	for i := 0; i < num; i++ {
		hllpp.Add(hashStr64(fmt.Sprintf("a", i)))
		hllpp.Add(hashStr64(fmt.Sprintf("a", i)))
	}
	elapsed = time.Since(start)
	fmt.Println("PP time elapsed:    ", elapsed)

	reg2 := 1 << reg
	e := float64(num) * 1.04 / math.Sqrt(float64(reg2))
	fmt.Printf("Should be between %f and %f\n", float64(num) - e, float64(num) + e)
	fmt.Printf("Other: %d\n", hll2.Count())
	fmt.Printf("Mine: %d\n", hll.Estimate())
	fmt.Printf("PP: %d\n", hllpp.Estimate())
}
