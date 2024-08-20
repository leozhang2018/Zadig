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
	"fmt"

	"github.com/koderover/zadig/v2/pkg/tool/httpclient"
	"github.com/koderover/zadig/v2/pkg/types"
)

func (c *Client) GetGroupDetailedInfo(groupID string) (*types.DetailedUserGroupResp, error) {
	url := fmt.Sprintf("/user-group/%s", groupID)
	resp := new(types.DetailedUserGroupResp)

	_, err := c.Get(url, httpclient.SetResult(&resp))

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) GetUserGroupsByUid(uid string) (*types.ListUserGroupResp, error) {
	url := "/user-group"
	resp := &types.ListUserGroupResp{}
	queries := make(map[string]string)
	queries["uid"] = uid

	_, err := c.Get(url, httpclient.SetQueryParams(queries), httpclient.SetResult(resp))
	return resp, err
}
