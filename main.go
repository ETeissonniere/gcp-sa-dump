package main

import (
	"flag"
)

func main() {
	outMode := flag.String("out", "text", "Output format (text, csv)")
	flag.Parse()

	if *outMode != "text" && *outMode != "csv" {
		panic("Invalid output mode")
	}

	results, err := runCollection()
	if err != nil {
		panic(err)
	}

	switch *outMode {
	case "text":
		outputText(results)
	case "csv":
		outputCSV(results)
	}
}
