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

	"github.com/gin-gonic/gin"
	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/workflow/service/workflow"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	"github.com/koderover/zadig/v2/pkg/tool/errors"
)

func ListPluginTemplates(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	ctx.Resp, ctx.Err = workflow.ListPluginTemplates(ctx.Logger)
}

func ListUnofficalPluginRepositories(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	ctx.Resp, ctx.Err = workflow.ListUnofficalPluginRepositories(ctx.Logger)
}

func DeletePluginRepo(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	// authorization check
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	ctx.Err = workflow.DeletePluginRepo(c.Param("id"), ctx.Logger)
}

func UpsertUserPluginRepository(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	// authorization check
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	req := new(commonmodels.PluginRepo)
	if err := c.ShouldBindJSON(req); err != nil {
		ctx.Err = errors.ErrInvalidParam.AddDesc(err.Error())
		return
	}
	ctx.Err = workflow.UpsertUserPluginRepository(req, ctx.Logger)
}

func UpsertEnterprisePluginRepository(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	req := new(commonmodels.PluginRepo)
	if err := c.ShouldBindJSON(req); err != nil {
		ctx.Err = errors.ErrInvalidParam.AddDesc(err.Error())
		return
	}
	ctx.Err = workflow.UpsertEnterprisePluginRepository(req, ctx.Logger)
}
