package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dannysecurity/bloomlab/bloom"
)

func main() {
	capacity := flag.Uint64("n", 1000, "expected number of items")
	fpr := flag.Float64("p", 0.01, "target false positive rate")
	remove := flag.Bool("remove", false, "remove words instead of adding")
	flag.Parse()

	cf, err := bloom.NewCountingFromTarget(*capacity, *fpr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingdemo: %v\n", err)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("Counting Bloom filter: m=%d counters, k=%d hashes\n", cf.BitCount(), cf.HashCount())
		fmt.Println("Usage: countingdemo [flags] <word> ...")
		os.Exit(0)
	}

	for _, word := range args {
		key := []byte(word)
		if *remove {
			if cf.Remove(key) {
				fmt.Printf("%q: removed\n", word)
			} else {
				fmt.Printf("%q: not present (remove skipped)\n", word)
			}
			continue
		}

		if cf.Contains(key) {
			fmt.Printf("%q: maybe present\n", word)
		} else if err := cf.Add(key); err != nil {
			fmt.Fprintf(os.Stderr, "add %q: %v\n", word, err)
		} else {
			fmt.Printf("%q: added\n", word)
		}
	}

	if !*remove && len(args) > 0 {
		fmt.Printf("checked: %s\n", strings.Join(args, ", "))
	}
}
