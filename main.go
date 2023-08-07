package main

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/iam/v1"
)

type taskResult struct {
	ProjectID string
	SaDumps   []*saAccountDump
}

func main() {
	ctx := context.Background()

	projects, err := listProjects(ctx)
	if err != nil {
		panic(err)
	}

	var wgProjects sync.WaitGroup
	errCh := make(chan error, len(projects))
	projectsCh := make(chan taskResult, len(projects))

	for _, project := range projects {
		wgProjects.Add(1)

		go func(p string) {
			defer wgProjects.Done()

			err := turnOnLoggingAPIIfNecessary(ctx, p)
			if err != nil {
				errCh <- err
				return
			}

			serviceAccounts, err := listServiceAccounts(ctx, p)
			if err != nil {
				errCh <- err
				return
			}

			var wgServiceAccounts sync.WaitGroup
			servicesErrCh := make(chan error, len(serviceAccounts))
			servicesCh := make(chan saAccountDump, len(serviceAccounts))

			for _, sa := range serviceAccounts {
				wgServiceAccounts.Add(1)

				go func(sa *iam.ServiceAccount) {
					defer wgServiceAccounts.Done()

					dump, err := inspectServiceAccount(ctx, p, sa)
					if err != nil {
						servicesErrCh <- err
						return
					}

					servicesCh <- *dump
				}(sa)
			}
			wgServiceAccounts.Wait()

			close(servicesErrCh)
			close(servicesCh)

			for e := range servicesErrCh {
				errCh <- e
			}

			allDumps := make([]*saAccountDump, 0)
			for s := range servicesCh {
				allDumps = append(allDumps, &s)
			}

			projectsCh <- taskResult{
				ProjectID: p,
				SaDumps:   allDumps,
			}
		}(project)
	}

	wgProjects.Wait()
	close(errCh)
	close(projectsCh)

	for e := range errCh {
		panic(e)
	}

	results := map[string][]*saAccountDump{}
	for result := range projectsCh {
		results[result.ProjectID] = result.SaDumps
	}

	for project, dump := range results {
		fmt.Printf("Project: %s\n", project)
		for _, d := range dump {
			fmt.Println("\tService Account:", d.AccountId)
			fmt.Println("\t\tDisplay Name:", d.DisplayName)
			fmt.Println("\t\tEmail:", d.Email)
			fmt.Println("\t\tDisabled:", d.Disabled)
			if d.Created != "" {
				fmt.Println("\t\tCreated:", d.Created)
			}
			fmt.Println("\t\tKeys:")
			for _, key := range d.Keys {
				fmt.Println("\t\t\t", key)
			}
		}
	}
}
