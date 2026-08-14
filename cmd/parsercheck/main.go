package main

import (
	"encoding/json"
	"fmt"
	"os"

	"opensca-ui/internal/scanner"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: parsercheck <report.json>")
		os.Exit(1)
	}
	rpt, err := scanner.ParseReport("test", os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(2)
	}
	out := struct {
		TotalComponents int                  `json:"totalComponents"`
		TotalVulns      int                  `json:"totalVulns"`
		SeverityCount   map[string]int       `json:"severityCount"`
		Warning         string               `json:"warning"`
		FirstThreeVulns []scanner.Vuln       `json:"firstThreeVulns"`
		FirstThreeComps []scanner.Component  `json:"firstThreeComps"`
	}{
		TotalComponents: rpt.TotalComponents,
		TotalVulns:      rpt.TotalVulns,
		SeverityCount:   rpt.SeverityCount,
		Warning:         rpt.Warning,
	}
	for _, c := range rpt.Components {
		if len(c.Vulns) > 0 {
			for _, v := range c.Vulns {
				if len(out.FirstThreeVulns) < 3 {
					out.FirstThreeVulns = append(out.FirstThreeVulns, v)
				}
			}
		}
		if len(out.FirstThreeComps) < 3 {
			out.FirstThreeComps = append(out.FirstThreeComps, c)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}