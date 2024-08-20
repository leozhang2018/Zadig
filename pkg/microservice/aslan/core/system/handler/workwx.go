/*
 * Copyright 2024 The KodeRover Authors.
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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service/workwx"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/system/service"
	"github.com/koderover/zadig/v2/pkg/setting"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
	"github.com/koderover/zadig/v2/pkg/tool/log"
)

type getWorkWXDepartmentReq struct {
	DepartmentID int `json:"department_id" form:"department_id"`
}

func GetWorkWxDepartment(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	req := new(getWorkWXDepartmentReq)
	err := c.ShouldBindQuery(&req)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddErr(err)
		return
	}

	appID := c.Param("id")

	ctx.Resp, ctx.Err = service.GetWorkWxDepartment(appID, req.DepartmentID, ctx.Logger)
}

type getWorkWXUsersReq struct {
	DepartmentID int `json:"department_id" form:"department_id"`
}

func GetWorkWxUsers(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	req := new(getWorkWXUsersReq)
	err := c.ShouldBindQuery(&req)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddErr(err)
		return
	}

	appID := c.Param("id")

	ctx.Resp, ctx.Err = service.GetWorkWxUsers(appID, req.DepartmentID, ctx.Logger)
}

type validateWorkWXCallbackReq struct {
	EchoString   string `json:"echostr"       form:"echostr"`
	MsgSignature string `json:"msg_signature" form:"msg_signature"`
	Nonce        string `json:"nonce"         form:"nonce"`
	Timestamp    string `json:"timestamp"     form:"timestamp"`
}

func ValidateWorkWXCallback(c *gin.Context) {
	// no defer response required
	ctx := internalhandler.NewContext(c)
	query := new(validateWorkWXCallbackReq)

	err := c.ShouldBindQuery(query)
	if err != nil {
		c.Set(setting.ResponseError, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	plainText, err := service.ValidateWorkWXWebhook(c.Param("id"), query.Timestamp, query.Nonce, query.EchoString, query.MsgSignature, ctx.Logger)
	if err != nil {
		c.Set(setting.ResponseError, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, plainText)
}

type workWXCallbackReq struct {
	MsgSignature string `json:"msg_signature" form:"msg_signature"`
	Nonce        string `json:"nonce"         form:"nonce"`
	Timestamp    string `json:"timestamp"     form:"timestamp"`
}

func WorkWXEventHandler(c *gin.Context) {
	query := new(workWXCallbackReq)

	err := c.ShouldBindQuery(query)
	if err != nil {
		c.Set(setting.ResponseError, err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	log.Infof("WorkWXEventHandler: New request url %s", c.Request.RequestURI)
	body, err := c.GetRawData()
	if err != nil {
		c.Set(setting.ResponseError, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	_, err = workwx.EventHandler(c.Param("id"), body, query.MsgSignature, query.Timestamp, query.Nonce)
	if err != nil {
		c.Set(setting.ResponseError, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
}
