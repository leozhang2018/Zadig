/*
Copyright 2021 The KodeRover Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jira

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Project ...
type Project struct {
	ID   string `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
}

// ProjectService ...
type ProjectService struct {
	client *Client
}

// ListProjects https://developer.atlassian.com/cloud/jira/platform/rest/#api-api-2-project-get
func (s *ProjectService) ListProjects() ([]*Project, error) {
	list := make([]*Project, 0)
	url := s.client.Host + "/rest/api/2/project"

	resp, err := s.client.R().Get(url)
	if err != nil {
		return nil, err
	}
	if resp.GetStatusCode()/100 != 2 {
		return nil, errors.Errorf("get unexpected status code %d, body: %s", resp.GetStatusCode(), resp.String())
	}
	if err = resp.UnmarshalJson(&list); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return list, nil
}

func (s *ProjectService) ListAllStatues(project string) ([]string, error) {
	resp, err := s.client.Issue.GetTypes(project)
	if err != nil {
		return nil, err
	}
	statuses := sets.NewString()
	for _, status := range resp {
		statuses.Insert(status.Status...)
	}
	return statuses.List(), nil
}

//// ListComponents ...
//func (s *ProjectService) ListComponents(projectKey string) ([]*Component, error) {
//
//	resp := make([]*Component, 0)
//
//	url := s.client.Host + "/rest/api/2/project/" + projectKey + "/components"
//
//	err := s.client.Conn.CallWithJson(context.Background(), &resp, "GET", url, "")
//
//	return resp, err
//}
