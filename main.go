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

var (
	yamlSeparator = []byte("\n---")
	docSeparator  = []byte(`\n############## separate here ##############\n`)
	flagset       = flag.NewFlagSet("", flag.ExitOnError)

	help bool
)

type object struct {
	data []byte
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
	scanner := bufio.NewScanner(f)
	scanner.Split(splitYAMLDocument)

	documents := make([]*object, 0)
	for scanner.Scan() {
		if i := bytes.IndexByte(scanner.Bytes(), '#'); i >= 0 {
			continue
		}
		d := &object{
			data: append([]byte(nil), scanner.Bytes()...),
		}
		documents = append(documents, d)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%w", err)
	}

	jsonDocuments := make([][]byte, 0)

	for _, doc := range documents {
		obj, err := yaml.YAMLToJSON(doc.data)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		jsonDocuments = append(jsonDocuments, obj)
	}

	var out bytes.Buffer
	out.Write(bytes.Join(jsonDocuments, []byte(",")))

	var buf bytes.Buffer
	if len(jsonDocuments) >= 2 {
		buf.Write([]byte("["))
		buf.Write(out.Bytes())
		buf.Write([]byte("]"))
	} else {
		buf.Write(out.Bytes())
	}

	var data interface{}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		return fmt.Errorf("%w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func splitYAMLDocument(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len(yamlSeparator)
	if i := bytes.Index(data, yamlSeparator); i >= 0 {
		// We have a potential document terminator
		i += sep
		after := data[i:]
		if len(after) == 0 {
			// we can't read any more characters
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func onStdin() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
