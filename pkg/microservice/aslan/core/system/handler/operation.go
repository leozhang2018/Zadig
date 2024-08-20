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

package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	models2 "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/system/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/system/service"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
)

func GetOperationLogs(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	status, err := strconv.Atoi(c.Query("status"))
	if err != nil {
		ctx.Err = e.ErrFindOperationLog.AddErr(err)
		return
	}

	perPage, err := strconv.Atoi(c.Query("per_page"))
	if err != nil {
		ctx.Err = e.ErrFindOperationLog.AddErr(err)
		return
	}

	page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		ctx.Err = e.ErrFindOperationLog.AddErr(err)
		return
	}

	args := &service.OperationLogArgs{
		Username:    c.Query("username"),
		ProductName: c.Query("projectName"),
		Function:    c.Query("function"),
		Status:      status,
		PerPage:     perPage,
		Page:        page,
	}

	if args.PerPage == 0 {
		args.PerPage = 50
	}

	if args.Page == 0 {
		args.Page = 1
	}

	resp, count, err := service.FindOperation(args, ctx.Logger)
	ctx.Resp = resp
	ctx.Err = err
	c.Writer.Header().Set("X-Total", strconv.Itoa(count))
}

func AddSystemOperationLog(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	args := new(models2.OperationLog)
	err = c.BindJSON(args)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid insertOperationLogs args")
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	ctx.Resp, ctx.Err = service.InsertOperation(args, ctx.Logger)
}

type updateOperationArgs struct {
	Status int `json:"status"`
}

func UpdateOperationLog(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	args := new(updateOperationArgs)
	err = c.BindJSON(args)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid insertOperationLogs args")
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	ctx.Err = service.UpdateOperation(c.Param("id"), args.Status, ctx.Logger)
}
