package main

import (
	"bytes"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/encoding/yaml"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func getClipboard() ([]byte, error) {
	cmd := exec.Command("xsel")
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running xsel: %v", err)
	}

	return b, nil
}

func setClipboard(input []byte) error {
	cmd := exec.Command("xsel", "-i", "-b")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = bytes.NewBuffer(input)
	return cmd.Run()
}

func main() {
	cb, err := getClipboard()
	if err != nil {
		log.Fatal(err)
	}

	r := &cue.Runtime{}

	i, err := yaml.Decode(r, "clipboard.yaml", cb)
	if err != nil {
		log.Fatalf("error parsing input: %v", err)
	}

	a := i.Value().Syntax(cue.Final(), cue.Concrete(true))
	b, err := format.Node(a, format.Simplify())
	if err != nil {
		log.Fatalf("error serializing cue: %v", err)
	}

	if err := setClipboard(b); err != nil {
		log.Fatalf("error setting clipboard: %v", err)
	}
}
