package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/wicanr2/atari-talos-ai-toolkit/protocol"
)

func main() {
	version := flag.Bool("version", false, "print version")
	flag.Parse()
	if *version {
		fmt.Printf("%s %s (%s)\n", protocol.Name, protocol.Version, protocol.Wire)
		return
	}
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serve(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	session := protocol.NewSession()
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	for scanner.Scan() {
		line := scanner.Bytes()
		request, err := protocol.Decode(line)
		if err != nil {
			if encodeErr := encoder.Encode(protocol.Invalid("", err.Error())); encodeErr != nil {
				return encodeErr
			}
			continue
		}
		response, quit := session.Handle(request)
		if err := encoder.Encode(response); err != nil {
			return err
		}
		if quit {
			return nil
		}
	}
	return scanner.Err()
}
