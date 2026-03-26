// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Command fhirlint validates FHIR R4 resources from files or stdin.
//
// Usage:
//
//	fhirlint patient.json observation.json
//	cat bundle.json | fhirlint -
//	fhirlint -profile http://example.org/StructureDefinition/us-core-patient *.json
//	fhirlint -format json patient.json   # output as JSON
//
// Exit codes:
//
//	0 — all resources valid
//	1 — validation errors found
//	2 — usage/IO error
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/caucehealth/gofhir/r4/resources"
	"github.com/caucehealth/gofhir/r4/validate"
)

var (
	profileURL  = flag.String("profile", "", "StructureDefinition URL to validate against")
	profileFile = flag.String("profile-file", "", "StructureDefinition JSON file to load")
	format      = flag.String("format", "text", "output format: text or json")
	quiet       = flag.Bool("q", false, "quiet mode — only show errors, not warnings")
	version     = flag.Bool("version", false, "print version and exit")
)

const appVersion = "0.1.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fhirlint [flags] <file...>\n\n")
		fmt.Fprintf(os.Stderr, "Validates FHIR R4 JSON resources. Use '-' to read from stdin.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *version {
		fmt.Printf("fhirlint %s\n", appVersion)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	// Build validator
	var registry *validate.ProfileRegistry
	if *profileURL != "" && *profileFile != "" {
		registry = validate.NewProfileRegistry()
		data, err := os.ReadFile(*profileFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: read profile: %v\n", err)
			os.Exit(2)
		}
		if err := registry.Load(json.RawMessage(data)); err != nil {
			fmt.Fprintf(os.Stderr, "error: load profile: %v\n", err)
			os.Exit(2)
		}
	}

	hasErrors := false
	var allResults []fileResult

	for _, arg := range args {
		files, err := resolveFiles(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}

		for _, file := range files {
			data, err := readFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", file, err)
				os.Exit(2)
			}

			result, err := validate.ValidateJSON(json.RawMessage(data))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", file, err)
				os.Exit(2)
			}

			// Profile validation if configured
			if registry != nil && *profileURL != "" {
				opts := []validate.Option{validate.WithProfile(registry, *profileURL)}
				v := validate.New(opts...)
				res, parseErr := resources.ParseResource(json.RawMessage(data))
				if parseErr == nil {
					profileResult := v.Validate(res)
					result.Issues = append(result.Issues, profileResult.Issues...)
				}
			}

			fr := fileResult{File: file, Result: result}
			allResults = append(allResults, fr)

			if result.HasErrors() {
				hasErrors = true
			}
		}
	}

	switch *format {
	case "json":
		outputJSON(allResults)
	default:
		outputText(allResults)
	}

	if hasErrors {
		os.Exit(1)
	}
}

type fileResult struct {
	File   string           `json:"file"`
	Result *validate.Result `json:"result"`
}

func resolveFiles(arg string) ([]string, error) {
	if arg == "-" {
		return []string{"-"}, nil
	}
	// Support glob patterns
	matches, err := filepath.Glob(arg)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no files matching %q", arg)
	}
	return matches, nil
}

func readFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func outputText(results []fileResult) {
	for _, fr := range results {
		issues := fr.Result.Issues
		if *quiet {
			issues = nil
			for _, i := range fr.Result.Issues {
				if i.Severity == validate.SeverityError {
					issues = append(issues, i)
				}
			}
		}

		if len(issues) == 0 {
			if !*quiet {
				fmt.Printf("%s: OK\n", fr.File)
			}
			continue
		}

		for _, issue := range issues {
			severity := strings.ToUpper(string(issue.Severity))
			path := issue.Path
			if path == "" {
				path = "(root)"
			}
			fmt.Printf("%s: %s [%s] %s: %s\n", fr.File, severity, issue.Code, path, issue.Message)
		}
	}
}

func outputJSON(results []fileResult) {
	type jsonIssue struct {
		Severity string `json:"severity"`
		Code     string `json:"code"`
		Path     string `json:"path,omitempty"`
		Message  string `json:"message"`
	}
	type jsonResult struct {
		File   string      `json:"file"`
		Valid  bool        `json:"valid"`
		Issues []jsonIssue `json:"issues,omitempty"`
	}

	var output []jsonResult
	for _, fr := range results {
		jr := jsonResult{
			File:  fr.File,
			Valid: !fr.Result.HasErrors(),
		}
		for _, issue := range fr.Result.Issues {
			if *quiet && issue.Severity != validate.SeverityError {
				continue
			}
			jr.Issues = append(jr.Issues, jsonIssue{
				Severity: string(issue.Severity),
				Code:     string(issue.Code),
				Path:     issue.Path,
				Message:  issue.Message,
			})
		}
		output = append(output, jr)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}
