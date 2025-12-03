package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
)

func main() {
	capacity := flag.Uint64("n", 10000, "expected number of items")
	fpr := flag.Float64("p", 0.01, "target false positive rate")
	flag.Parse()

	f, err := bloom.New(*capacity, *fpr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bloomdemo: %v\n", err)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("Bloom filter: m=%d bits, k=%d hashes\n", f.BitCount(), f.HashCount())
		fmt.Println("Usage: bloomdemo [flags] <word> ...")
		os.Exit(0)
	}

	for _, word := range args {
		key := []byte(word)
		if f.Contains(key) {
			fmt.Printf("%q: maybe present\n", word)
		} else {
			f.Add(key)
			fmt.Printf("%q: added (was absent)\n", word)
		}
	}

	fmt.Printf("fill ratio: %.2f%%, inserts: %d\n", f.FillRatio()*100, f.ApproximateCount())
}
