package main

import (
	"fmt"
)

func outputText(results collectionResult) {
	for project, dump := range results {
		fmt.Printf("Project: %s\n", project)
		for _, d := range dump {
			fmt.Println("\tService Account:", d.AccountId)
			fmt.Println("\t\tDisplay Name:", d.DisplayName)
			fmt.Println("\t\tEmail:", d.Email)
			fmt.Println("\t\tDisabled:", d.Disabled)
			fmt.Println("\t\tKeys:")
			for _, key := range d.Keys {
				fmt.Println("\t\t\t", key)
			}
		}
	}
}
