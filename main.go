package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()

	projects, err := listProjects(ctx)
	if err != nil {
		panic(err)
	}

	for _, project := range projects {
		fmt.Printf("Project: %s\n", project)

		err = turnOnLoggingAPIIfNecessary(ctx, project)
		if err != nil {
			panic(err)
		}

		serviceAccounts, err := listServiceAccounts(ctx, project)
		if err != nil {
			panic(err)
		}

		for _, sa := range serviceAccounts {
			dump, err := inspectServiceAccount(ctx, project, sa)
			if err != nil {
				panic(err)
			}

			fmt.Println("\tService Account:", dump.AccountId)
			fmt.Println("\t\tDisplay Name:", dump.DisplayName)
			fmt.Println("\t\tEmail:", dump.Email)
			fmt.Println("\t\tDisabled:", dump.Disabled)
			if dump.Created != "" {
				fmt.Println("\t\tCreated:", dump.Created)
			}
			fmt.Println("\t\tKeys:")
			for _, key := range dump.Keys {
				fmt.Println("\t\t\t", key)
			}
		}
	}
}
