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
	"github.com/gin-gonic/gin/binding"

	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	deliveryservice "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/delivery/service"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
)

func ListDeliveryArtifacts(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if !ctx.Resources.SystemActions.DeliveryCenter.ViewArtifact {
			ctx.UnAuthorized = true
			return
		}
	}

	args := new(commonrepo.DeliveryArtifactArgs)
	args.Type = c.Query("type")
	args.Image = c.Query("image")
	args.Name = c.Query("name")
	args.ImageTag = c.Query("imageTag")
	args.RepoName = c.Query("repoName")
	args.Branch = c.Query("branch")
	args.Source = c.Query("source")

	perPageStr := c.Query("per_page")
	pageStr := c.Query("page")
	var (
		perPage int
		page    int
	)
	if perPageStr == "" {
		perPage = 20
	} else {
		perPage, err = strconv.Atoi(perPageStr)
		if err != nil {
			ctx.Err = e.ErrInvalidParam.AddDesc(fmt.Sprintf("perPage args err :%s", err))
			return
		}
	}

	if pageStr == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			ctx.Err = e.ErrInvalidParam.AddDesc(fmt.Sprintf("page args err :%s", err))
			return
		}
	}
	args.PerPage = perPage
	args.Page = page

	artifacts, total, err := deliveryservice.ListDeliveryArtifacts(args, ctx.Logger)
	c.Writer.Header().Add("X-Total", strconv.Itoa(total))
	ctx.Resp, ctx.Err = artifacts, err
}

func GetDeliveryArtifactIDByImage(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if !ctx.Resources.SystemActions.DeliveryCenter.ViewArtifact {
			ctx.UnAuthorized = true
			return
		}
	}

	image := c.Query("image")
	if image == "" {
		ctx.Err = e.ErrInvalidParam.AddDesc("image can't be empty!")
		return
	}
	args := &commonrepo.DeliveryArtifactArgs{
		Image: image,
	}

	ctx.Resp, ctx.Err = deliveryservice.GetDeliveryArtifactIDByImage(args, ctx.Logger)
}

func GetDeliveryArtifact(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if !ctx.Resources.SystemActions.DeliveryCenter.ViewArtifact {
			ctx.UnAuthorized = true
			return
		}
	}

	id := c.Param("id")
	if id == "" {
		ctx.Err = e.ErrInvalidParam.AddDesc("id can't be empty!")
		return
	}
	args := &commonrepo.DeliveryArtifactArgs{
		ID: id,
	}

	ctx.Resp, ctx.Err = deliveryservice.GetDeliveryArtifact(args, ctx.Logger)
}

func CreateDeliveryActivities(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {

		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if !ctx.Resources.SystemActions.DeliveryCenter.ViewArtifact {
			ctx.UnAuthorized = true
			return
		}
	}

	var deliveryActivity commonmodels.DeliveryActivity
	if err := c.ShouldBindWith(&deliveryActivity, binding.JSON); err != nil {
		ctx.Logger.Infof("ShouldBindWith err :%v", err)
		ctx.Err = e.ErrInvalidParam.AddDesc(err.Error())
		return
	}
	ID := c.Param("id")
	if ID == "" {
		ctx.Err = e.ErrInvalidParam.AddDesc("id can't be empty!")
		return
	}
	ctx.Err = deliveryservice.InsertDeliveryActivities(&deliveryActivity, ID, ctx.Logger)
}
