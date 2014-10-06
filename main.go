package main

import (
	"math/rand"
	"math"
	"fmt"
	"hash"
	"hash/fnv"
)

func hash32(s string) hash.Hash32 {
	h := fnv.New32()
	h.Write([]byte(s))
	return h
}

func hash64(s string) hash.Hash64 {
	h := fnv.New64()
	h.Write([]byte(s))
	return h
}

func randStr(n int64) string {
	i := rand.Uint32()
	return fmt.Sprintf("a%s %s", i, n)
}

func main() {
	tot_err := 0.0
	tot_errpp := 0.0
	runs := 0

	for p := uint8(14); p < 15; p += 2 {
		h, _ := NewHyperLogLog(p)
		hpp, _ := NewHyperLogLogPP(p)

		for n := int64(1000); n < 20000; n += 100 {
			for i := int64(0); i < n; i++ {
				s := randStr(i)
				h.Add(hash32(s))
				h.Add(hash32(s))
				hpp.Add(hash64(s))
				hpp.Add(hash64(s))
			}

			e := h.Estimate()
			epp := hpp.Estimate()

			runs++
			err := n - int64(e)
			errpp := n - int64(epp)

			err_perc := math.Abs(float64(err) / float64(n))
			errpp_perc := math.Abs(float64(errpp) / float64(n))
			tot_err += err_perc
			tot_errpp += errpp_perc

			h.Clear()
			hpp.Clear()

			fmt.Printf("Precision: %d, N: %d\n", p, n)
			fmt.Printf("  HLL  : %d, Error: %d, %%: %f\n", e, err, err_perc)
			fmt.Printf("  HLLPP: %d, Error: %d, %%: %f\n\n", epp, errpp, errpp_perc)
		}
	}
	fmt.Printf("HLL Total Err %%  : %f\n", tot_err / float64(runs))
	fmt.Printf("HLLPP Total Err %%: %f\n", tot_errpp / float64(runs))
}
