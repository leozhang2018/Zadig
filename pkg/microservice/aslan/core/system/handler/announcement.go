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

	"github.com/gin-gonic/gin"
	systemmodel "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/system/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/system/service"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
)

func CreateAnnouncement(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	args := new(systemmodel.Announcement)

	err = c.BindJSON(args)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid notify args")
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	ctx.Err = service.CreateAnnouncement(ctx.UserName, args, ctx.Logger)
}

func UpdateAnnouncement(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	args := new(systemmodel.Announcement)

	err = c.BindJSON(args)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid notify args")
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	ctx.Err = service.UpdateAnnouncement(ctx.UserName, args.ID.Hex(), args, ctx.Logger)
}

func PullAllAnnouncement(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	ctx.Resp, ctx.Err = service.PullAllAnnouncement(ctx.UserName, ctx.Logger)
}

func PullNotifyAnnouncement(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	ctx.Resp, ctx.Err = service.PullNotifyAnnouncement(ctx.UserName, ctx.Logger)
}

func DeleteAnnouncement(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	ID := c.Param("id")

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		ctx.UnAuthorized = true
		return
	}

	ctx.Err = service.DeleteAnnouncement(ctx.UserName, ID, ctx.Logger)
}
