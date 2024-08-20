/*
Copyright 2022 The KodeRover Authors.

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

package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	commonutil "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/util"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/environment/service"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
	"github.com/koderover/zadig/v2/pkg/types"
)

func ConnectSshPmExec(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	projectKey := c.Query("projectName")
	ip := c.Query("ip")
	hostId := c.Query("hostId")
	name := c.Param("name")
	if projectKey == "" || ip == "" || name == "" || hostId == "" {
		ctx.Err = e.ErrInvalidParam.AddDesc("param projectName or ip or name or hostId is empty")
	}
	colsStr := c.DefaultQuery("cols", "135")
	cols, err := strconv.Atoi(colsStr)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddErr(err)
	}
	rowsStr := c.DefaultQuery("rows", "40")
	rows, err := strconv.Atoi(rowsStr)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddErr(err)
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[projectKey]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !ctx.Resources.ProjectAuthInfo[projectKey].IsProjectAdmin &&
			!ctx.Resources.ProjectAuthInfo[projectKey].Env.SSH {
			permitted, err := internalhandler.GetCollaborationModePermission(ctx.UserID, projectKey, types.ResourceTypeEnvironment, name, types.EnvActionSSH)
			if err != nil || !permitted {
				ctx.UnAuthorized = true
				return
			}
		}
	}

	ctx.Err = service.ConnectSshPmExec(c, ctx.UserName, name, projectKey, ip, hostId, cols, rows, ctx.Logger)
}

// @summary Exec VM Service Command
// @description Exec VM Service Command
// @Tags 	environment
// @accept 	json
// @produce json
// @Param 	projectName		query		string									true	"project name"
// @Param 	hostId			query		string									true	"host id"
// @Param 	commandType		query		service.VmServiceCommandType			true	"vm service command type"
// @Param 	name 			path		string									true	"env name"
// @Param 	serviceName 	path		string									true	"serivce name"
// @success 200 			{object} 	service.ExecVmServiceCommandResponse
// @Router /api/aslan/environment/environments/{name}/services/{serviceName}/execmd [post]
func ExecVmServiceCommand(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	projectKey := c.Query("projectName")
	hostId := c.Query("hostId")
	commandType := c.Query("commandType")
	name := c.Param("name")
	serviceName := c.Param("serviceName")
	if projectKey == "" || name == "" || hostId == "" || commandType == "" {
		ctx.Err = e.ErrInvalidParam.AddDesc("param projectName or name or hostId or commandType is empty")
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[projectKey]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !ctx.Resources.ProjectAuthInfo[projectKey].IsProjectAdmin &&
			!ctx.Resources.ProjectAuthInfo[projectKey].Env.ManagePods {
			permitted, err := internalhandler.GetCollaborationModePermission(ctx.UserID, projectKey, types.ResourceTypeEnvironment, name, types.EnvActionManagePod)
			if err != nil || !permitted {
				ctx.UnAuthorized = true
				return
			}
		}
	}

	err = commonutil.CheckZadigProfessionalLicense()
	if err != nil {
		ctx.Err = err
		return
	}

	ctx.Resp, ctx.Err = service.ExecVmServiceCommand(projectKey, name, serviceName, hostId, service.VmServiceCommandType(commandType), ctx.Logger)
}
