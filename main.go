package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/serviceusage/v1"
)

func main() {
	ctx := context.Background()

	// List projects
	crmService, _ := cloudresourcemanager.NewService(ctx)
	projectList, _ := crmService.Projects.List().Do()

	for _, project := range projectList.Projects {
		fmt.Printf("Project: %s\n", project.ProjectId)

		// Check if the Logging API is enabled
		serviceUsageService, _ := serviceusage.NewService(ctx)
		enabledAPIs, _ := serviceUsageService.Services.List("projects/" + project.ProjectId).Filter("state:ENABLED").Do()

		loggingEnabled := false
		for _, service := range enabledAPIs.Services {
			if service.Name == "logging.googleapis.com" {
				loggingEnabled = true
				break
			}
		}

		if !loggingEnabled {
			serviceUsageService.Services.Enable("projects/"+project.ProjectId+"/services/logging.googleapis.com", &serviceusage.EnableServiceRequest{}).Do()
			time.Sleep(10 * time.Second)
		}

		// Service accounts
		iamService, _ := iam.NewService(ctx)
		serviceAccounts, _ := iamService.Projects.ServiceAccounts.List("projects/" + project.ProjectId).Do()

		for _, sa := range serviceAccounts.Accounts {
			accountId := strings.Split(sa.Email, "@")[0]
			fmt.Printf("  Service Account: %s\n", accountId)
			fmt.Printf("    DisplayName: %s\n", sa.DisplayName)
			fmt.Printf("    Disabled: %v\n", sa.Disabled)
			fmt.Printf("    Email: %s\n", sa.Email)

			// Keys
			keys, _ := iamService.Projects.ServiceAccounts.Keys.List("projects/" + project.ProjectId + "/serviceAccounts/" + sa.Email).Do()
			for _, key := range keys.Keys {
				fmt.Printf("    Key: %s\n", key.Name)
			}

			// Logs
			loggingService, _ := logging.NewService(ctx)
			logName := fmt.Sprintf("projects/%s/logs/cloudaudit.googleapis.com%%2Factivity", project.ProjectId)
			filter := fmt.Sprintf(
				`logName="%s" resource.type="service_account" protoPayload.methodName="google.iam.admin.v1.CreateServiceAccount" protoPayload.request.account_id="%s"`,
				logName, accountId,
			)
			logEntries, err := loggingService.Entries.List(&logging.ListLogEntriesRequest{
				ResourceNames: []string{"projects/" + project.ProjectId},
				Filter:        filter,
				OrderBy:       "timestamp desc",
			}).Do()
			if err != nil {
				fmt.Println(err)
				continue
			}

			var logTime string
			if len(logEntries.Entries) > 0 {
				logTime = logEntries.Entries[0].Timestamp
			}

			if logTime == "" {
				fmt.Println("    Created: Not Found")
			} else {
				fmt.Printf("    Created: %s\n", logTime)
			}
		}
	}
}
