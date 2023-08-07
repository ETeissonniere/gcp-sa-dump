package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/schollz/progressbar/v3"
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

	status := fmt.Sprintf("scanning %d projects", len(projects))
	bar := progressbar.Default(int64(len(projects)), status)

	var wgProjects sync.WaitGroup
	errCh := make(chan error, len(projects))
	projectsCh := make(chan taskResult, len(projects))

	for _, project := range projects {
		wgProjects.Add(1)

		go func(p string) {
			defer wgProjects.Done()

			serviceAccounts, err := listServiceAccounts(ctx, p)
			if err != nil {
				errCh <- err
				return
			}

			bar.Describe(fmt.Sprintf("%s (%d service accounts)", p, len(serviceAccounts)))

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

			bar.Add(1)
			bar.Describe(fmt.Sprintf("completed %s", p))

			allErrors := make([]error, 0)
			for e := range servicesErrCh {
				allErrors = append(allErrors, e)
			}

			if len(allErrors) > 0 {
				errCh <- fmt.Errorf("Error processing project %s: %v", p, allErrors)
				return
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

	bar.Finish()

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
			fmt.Println("\t\tKeys:")
			for _, key := range d.Keys {
				fmt.Println("\t\t\t", key)
			}
		}
	}
}
