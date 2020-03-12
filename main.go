package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"sigs.k8s.io/yaml"
)

var flagset = flag.NewFlagSet("", flag.ExitOnError)

var help bool

func init() {
	flagset.BoolVar(&help, "h", false, "show help message")
}

func main() {
	if err := flagset.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [FILENAME]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  cat foo.yaml | %s\n\n", os.Args[0])
		flagset.PrintDefaults()
	}

	if help {
		flagset.Usage()
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		if !onStdin() {
			flagset.Usage()
			os.Exit(0)
		}
		if err := convertYAMLToJSON(os.Stdin); err != nil {
			log.Fatal(err)
		}
	} else {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}

		if err := convertYAMLToJSON(f); err != nil {
			log.Fatal(err)
		}
	}
}

func convertYAMLToJSON(f *os.File) error {
	var out bytes.Buffer

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		out.Write(scanner.Bytes())
		out.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%w", err)
	}

	jsonObj, err := yaml.YAMLToJSON(out.Bytes())
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	out.Reset() // reset the buffer so we can put our json in there
	json.Indent(&out, jsonObj, "", "  ")

	fmt.Fprint(os.Stdout, out.String())

	return nil
}

func onStdin() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
