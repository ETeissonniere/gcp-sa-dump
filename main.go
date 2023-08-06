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

func listProjects(ctx context.Context) ([]string, error) {
	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, err
	}

	projectList, err := crmService.Projects.List().Do()
	if err != nil {
		return nil, err
	}

	var projectIds []string
	for _, project := range projectList.Projects {
		projectIds = append(projectIds, project.ProjectId)
	}

	return projectIds, nil
}

func turnOnLoggingAPIIfNecessary(ctx context.Context, projectId string) {
	serviceUsageService, _ := serviceusage.NewService(ctx)
	enabledAPIs, _ := serviceUsageService.Services.List("projects/" + projectId).Filter("state:ENABLED").Do()

	loggingEnabled := false
	for _, service := range enabledAPIs.Services {
		splitted := strings.Split(service.Name, "/")
		if splitted[len(splitted)-1] == "logging.googleapis.com" {
			loggingEnabled = true
			break
		}
	}

	if !loggingEnabled {
		serviceUsageService.Services.Enable("projects/"+projectId+"/services/logging.googleapis.com", &serviceusage.EnableServiceRequest{}).Do()
		time.Sleep(10 * time.Second) // give GCP some time to actually turn the API on
	}
}

func listServiceAccounts(ctx context.Context, projectId string) ([]*iam.ServiceAccount, error) {
	iamService, _ := iam.NewService(ctx)
	serviceAccounts, err := iamService.Projects.ServiceAccounts.List("projects/" + projectId).Do()
	if err != nil {
		return nil, err
	}

	return serviceAccounts.Accounts, nil
}

func inspectServiceAccount(ctx context.Context, projectId string, sa *iam.ServiceAccount) {
	iamService, _ := iam.NewService(ctx)

	accountId := strings.Split(sa.Email, "@")[0]
	fmt.Printf("  Service Account: %s\n", accountId)
	fmt.Printf("    DisplayName: %s\n", sa.DisplayName)
	fmt.Printf("    Disabled: %v\n", sa.Disabled)
	fmt.Printf("    Email: %s\n", sa.Email)

	// Keys
	keys, _ := iamService.Projects.ServiceAccounts.Keys.List("projects/" + projectId + "/serviceAccounts/" + sa.Email).Do()
	for _, key := range keys.Keys {
		fmt.Printf("    Key: %s\n", key.Name)
	}

	// Logs
	loggingService, _ := logging.NewService(ctx)
	logName := fmt.Sprintf("projects/%s/logs/cloudaudit.googleapis.com%%2Factivity", projectId)
	filter := fmt.Sprintf(
		`logName="%s" resource.type="service_account" protoPayload.methodName="google.iam.admin.v1.CreateServiceAccount" protoPayload.request.account_id="%s"`,
		logName, accountId,
	)
	logEntries, err := loggingService.Entries.List(&logging.ListLogEntriesRequest{
		ResourceNames: []string{"projects/" + projectId},
		Filter:        filter,
		OrderBy:       "timestamp desc",
	}).Do()
	if err != nil {
		fmt.Println(err)
		return
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

func main() {
	ctx := context.Background()

	projects, _ := listProjects(ctx)

	for _, project := range projects {
		fmt.Printf("Project: %s\n", project)

		turnOnLoggingAPIIfNecessary(ctx, project)
		serviceAccounts, _ := listServiceAccounts(ctx, project)

		for _, sa := range serviceAccounts {
			inspectServiceAccount(ctx, project, sa)
		}
	}
}
