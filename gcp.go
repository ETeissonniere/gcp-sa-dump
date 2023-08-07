package main

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
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

	return ret, nil
}
