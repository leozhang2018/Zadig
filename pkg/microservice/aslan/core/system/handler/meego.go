/*
 * Copyright 2022 The KodeRover Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/system/service"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
)

// @Summary List Meego Projects
// @Description List Meego Projects
// @Tags 	system
// @Accept 	json
// @Produce json
// @Param 	id 		path		string										true	"meego id"
// @Success 200 	{object} 	service.MeegoProjectResp
// @Router /api/aslan/system/meego/{id}/projects [get]
func GetMeegoProjects(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	id := c.Param("id")
	ctx.Resp, ctx.Err = service.GetMeegoProjects(id)
}

// @Summary Get Meego Work Item Type List
// @Description Get Meego Work Item Type List
// @Tags 	system
// @Accept 	json
// @Produce json
// @Param 	id 		        path		string										true	"meego id"
// @Param 	projectID 		path		string										true	"project id"
// @Success 200 			{object} 	service.MeegoWorkItemTypeResp
// @Router /api/aslan/system/meego/{id}/projects/{projectID}/work_item/types [get]
func GetWorkItemTypeList(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	id := c.Param("id")
	projectID := c.Param("projectID")

	ctx.Resp, ctx.Err = service.GetWorkItemTypeList(id, projectID)
}

// @Summary List Meego Work Items
// @Description List Meego Work Items
// @Tags 	system
// @Accept 	json
// @Produce json
// @Param 	id 		        path		string										true	"meego id"
// @Param 	projectID 		path		string										true	"project id"
// @Param 	type_key 		query		string										true	"type key"
// @Param 	page_num 		query		string										true	"page num"
// @Param 	page_size 		query		string										true	"page size"
// @Param 	item_name 		query		string										true	"item name"
// @Success 200 			{object} 	service.MeegoWorkItemResp
// @Router /api/aslan/system/meego/{id}/projects/{projectID}/work_item [get]
func ListMeegoWorkItems(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	projectID := c.Param("projectID")
	typeKey := c.Query("type_key")
	pageNumStr := c.Query("page_num")
	pageNum := 0
	if pageNumStr != "" {
		pageNumber, err := strconv.Atoi(pageNumStr)
		if err != nil {
			ctx.Err = errors.New("invalid page_num")
			return
		}
		pageNum = pageNumber
	}
	pageSizeStr := c.Query("page_size")
	pageSize := 0
	if pageSizeStr != "" {
		pageSizeConv, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			ctx.Err = errors.New("invalid page_size")
			return
		}
		pageSize = pageSizeConv
	}
	if typeKey == "" {
		ctx.Err = errors.New("type_key cannot be empty")
		return
	}

	id := c.Param("id")
	nameQuery := c.Query("item_name")

	ctx.Resp, ctx.Err = service.ListMeegoWorkItems(id, projectID, typeKey, nameQuery, pageNum, pageSize)
}

func ListAvailableWorkItemTransitions(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	projectID := c.Param("projectID")
	workItemIDStr := c.Param("workItemID")
	workItemID, err := strconv.Atoi(workItemIDStr)
	if err != nil {
		ctx.Err = errors.New("invalid work item id")
		return
	}
	workItemTypeKey := c.Query("type_key")

	id := c.Param("id")
	ctx.Resp, ctx.Err = service.ListAvailableWorkItemTransitions(id, projectID, workItemTypeKey, workItemID)
}
