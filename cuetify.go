package main

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/encoding/yaml"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

var (
	flagPackage = flag.String("p", "components", "Package name for generated Cue files")
	flagOutDir  = flag.String("o", ".", "Output directory for generated Cue files")
)

func init() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: cuetify [flags] FILENAME...\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	for _, filename := range flag.Args() {
		log.Printf("Processing %s", filename)
		if err := process(filename); err != nil {
			log.Fatalf("Error processing %s: %v", filename, err)
		}
	}
}

func process(filename string) error {
	r := &cue.Runtime{}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	i, err := yaml.Decode(r, filename, f)
	if err != nil {
		return err
	}
	iter, err := i.Value().List()
	if err != nil {
		return err
	}

	objects := make(map[string]*cue.Instance)

	for iter.Next() {
		v := iter.Value()

		apiVersion, err := v.LookupPath(cue.MakePath(cue.Str("apiVersion"))).String()
		if err != nil {
			return fmt.Errorf("failed to parse apiVersion: %v", err)
		}
		kind, err := v.LookupPath(cue.MakePath(cue.Str("kind"))).String()
		if err != nil {
			return fmt.Errorf("failed to parse kind: %v", err)
		}
		name, err := v.LookupPath(cue.MakePath(cue.Str("metadata"), cue.Str("name"))).String()
		if err != nil {
			return fmt.Errorf("failed to parse metadata.name: %v", err)
		}

		log.Printf("Found %s %s %s", apiVersion, kind, name)

		group := pluralize(kind)
		out, ok := objects[group]
		if !ok {
			out, err = r.Compile("empty.cue", "{}")
			if err != nil {
				panic(err)
			}
			objects[group] = out
		}

		objects[group], err = out.Fill(v, "k8s", group, name)
		if err != nil {
			return err
		}
	}

	for group, inst := range objects {
		decls := inst.Value().Syntax(cue.Final(), cue.Concrete(true)).(*ast.StructLit).Elts
		af := &ast.File{Decls: []ast.Decl{
			&ast.CommentGroup{List: []*ast.Comment{
				{Text: "// cuetify import from " + filename},
			}},
			&ast.Package{Name: ast.NewIdent(*flagPackage)},
		}}
		af.Decls = append(af.Decls, decls...)

		b, err := format.Node(af, format.Simplify())
		if err != nil {
			return err
		}

		filename := path.Join(*flagOutDir, fmt.Sprintf("%s.cue", group))
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
		if err != nil {
			log.Printf("failed to open %s: %v", filename, err)
			continue
		}
		if _, err = f.Write(b); err != nil {
			log.Printf("failed to write to %s: %v", filename, err)
			continue
		}
		if err = f.Close(); err != nil {
			log.Printf("failed to close %s: %v", filename, err)
			continue
		}

		log.Printf("Wrote %s", filename)
	}

	return nil
}

func pluralize(kind string) string {
	switch kind {
	case "PersistentVolumeClaim":
		return "pvcs"
	case "CustomResourceDefinition":
		return "crds"
	default:
		return strings.ToLower(kind) + "s"
	}
}
