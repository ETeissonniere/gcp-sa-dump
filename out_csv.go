package main

import (
	"fmt"
)

func outputCSV(results collectionResult) {
	fmt.Println("Project,Service Account,Display Name,Email,Disabled,Key")

	for project, dump := range results {
		for _, d := range dump {
			for _, k := range d.Keys {
				fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\",\"%t\",\"%s\"\n", project, d.AccountId, d.DisplayName, d.Email, d.Disabled, k)
			}
		}
	}
}
