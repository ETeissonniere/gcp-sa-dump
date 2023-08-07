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
		return nil, fmt.Errorf("Error creating CloudResourceManager service: %v", err)
	}

	projectList, err := crmService.Projects.List().Do()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving projects list: %v", err)
	}

	var projectIds []string
	for _, project := range projectList.Projects {
		if project.LifecycleState == "ACTIVE" {
			projectIds = append(projectIds, project.ProjectId)
		}
	}

	return projectIds, nil
}

func turnOnLoggingAPIIfNecessary(ctx context.Context, projectId string) error {
	serviceUsageService, err := serviceusage.NewService(ctx)
	if err != nil {
		return fmt.Errorf("Error creating ServiceUsage service: %v", err)
	}
	enabledAPIs, err := serviceUsageService.Services.List("projects/" + projectId).Filter("state:ENABLED").Do()
	if err != nil {
		return fmt.Errorf("Error retrieving enabled APIs: %v", err)
	}

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

	return nil
}

func listServiceAccounts(ctx context.Context, projectId string) ([]*iam.ServiceAccount, error) {
	iamService, _ := iam.NewService(ctx)
	serviceAccounts, err := iamService.Projects.ServiceAccounts.List("projects/" + projectId).Do()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving service accounts list: %v", err)
	}

	return serviceAccounts.Accounts, nil
}

type saAccountDump struct {
	AccountId   string
	DisplayName string
	Email       string
	Disabled    bool
	Keys        []string
	Created     string
}

func inspectServiceAccount(ctx context.Context, projectId string, sa *iam.ServiceAccount) (*saAccountDump, error) {
	iamService, err := iam.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error creating IAM service: %v", err)
	}

	ret := &saAccountDump{
		AccountId:   strings.Split(sa.Email, "@")[0],
		DisplayName: sa.DisplayName,
		Email:       sa.Email,
		Disabled:    sa.Disabled,
		Keys:        make([]string, 0),
	}

	// Keys
	keys, err := iamService.Projects.ServiceAccounts.Keys.List("projects/" + projectId + "/serviceAccounts/" + sa.Email).Do()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving service account keys: %v", err)
	}
	for _, key := range keys.Keys {
		ret.Keys = append(ret.Keys, key.Name)
	}

	// Logs
	loggingService, err := logging.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error creating Logging service: %v", err)
	}
	logName := fmt.Sprintf("projects/%s/logs/cloudaudit.googleapis.com%%2Factivity", projectId)
	filter := fmt.Sprintf(
		`logName="%s" resource.type="service_account" protoPayload.methodName="google.iam.admin.v1.CreateServiceAccount" protoPayload.request.account_id="%s"`,
		logName, ret.AccountId,
	)
	logEntries, err := loggingService.Entries.List(&logging.ListLogEntriesRequest{
		ResourceNames: []string{"projects/" + projectId},
		Filter:        filter,
		OrderBy:       "timestamp desc",
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving service account logs: %v", err)
	}

	var logTime string
	if len(logEntries.Entries) > 0 {
		logTime = logEntries.Entries[0].Timestamp
	}

	if logTime != "" {
		ret.Created = logTime
	}

	return ret, nil
}
