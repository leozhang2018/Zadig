/*
Copyright 2023 The KodeRover Authors.

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

package user

import (
	"strconv"

	"github.com/koderover/zadig/v2/pkg/tool/httpclient"
)

type InitializeProjectResp struct {
	Roles []string `json:"roles"`
}

func (c *Client) InitializeProject(projectKey string, isPublic bool, admins []string) error {
	url := "policy/internal/initializeProject"

	body := map[string]interface{}{
		"namespace": projectKey,
		"is_public": isPublic,
		"admins":    admins,
	}

	_, err := c.Post(url, httpclient.SetBody(body))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) SetProjectVisibility(namespace string, isVisible bool) error {
	url := "policy/internal/setProjectVisibility"

	query := map[string]string{
		"namespace": namespace,
		"is_public": strconv.FormatBool(isVisible),
	}

	_, err := c.Post(url, httpclient.SetQueryParams(query))
	if err != nil {
		return err
	}
	return nil
}
