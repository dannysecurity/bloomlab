package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
)

func main() {
	flags := filterflags.Register(10_000)
	flag.Parse()

	cfg, err := flags.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bloomdemo: %v\n", err)
		os.Exit(1)
	}
	f, err := bloom.NewFilter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bloomdemo: %v\n", err)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println(cfg.String())
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

	fmt.Printf("fill ratio: %.2f%%, inserts: %d, theoretical FPR: %.4f%%\n",
		f.FillRatio()*100, f.ApproximateCount(), f.TheoryFPR()*100)
}
