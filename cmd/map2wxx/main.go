// Copyright (c) 2026 Michael D Henderson. All rights reserved.

// Command map2wxx converts an hmz2map hex map into a Worldographer map.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/maloquacious/hmz2map"
	"github.com/maloquacious/map2wxx"
	"github.com/maloquacious/wxx/xmlio"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "map2wxx: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("map2wxx", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: map2wxx [flags] <map.json>\n\n")
		fs.PrintDefaults()
	}
	output := fs.String("output", "", "Worldographer .wxx file to write (required)")
	app := fs.String("app", map2wxx.AppCurrent, fmt.Sprintf("Worldographer application `version` to write, or %q for %s", map2wxx.AppCurrent, xmlio.CurrentApp()))
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, map2wxx.Version())
		return nil
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected one input file, got %d", fs.NArg())
	}
	if *output == "" {
		return fmt.Errorf("-output is required")
	}
	appVersion := map2wxx.ResolveApp(*app)
	if err := map2wxx.CheckApp(appVersion); err != nil {
		return fmt.Errorf("-app %s: %w", *app, err)
	}
	input := fs.Arg(0)
	start := time.Now()
	phase := func(name string) {
		fmt.Fprintf(stdout, "%-24s %6.1fs\n", name, time.Since(start).Seconds())
	}

	var m hmz2map.Map
	if err := readJSON(input, &m); err != nil {
		return err
	}
	phase("read input")
	w, err := map2wxx.Convert(&m, appVersion)
	if err != nil {
		return fmt.Errorf("%s: %w", input, err)
	}
	phase("convert")
	if err := xmlio.WriteFile(*output, w, appVersion); err != nil {
		return err
	}
	phase("write output")

	fmt.Fprintf(stdout, "map:              %d × %d, border %d\n", m.Columns, m.Rows, m.Border)
	fmt.Fprintf(stdout, "worldographer:    %s, COLUMNS\n", appVersion)
	return nil
}

func readJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(bufio.NewReader(f)).Decode(v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
