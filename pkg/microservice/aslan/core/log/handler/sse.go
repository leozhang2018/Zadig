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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service/workflowcontroller/jobcontroller"
	logservice "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/log/service"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/workflow/testing/service"
	"github.com/koderover/zadig/v2/pkg/setting"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
	"github.com/koderover/zadig/v2/pkg/types"
	"github.com/koderover/zadig/v2/pkg/util/ginzap"
)

func GetContainerLogsSSE(c *gin.Context) {
	logger := ginzap.WithContext(c).Sugar()

	ctx, err := internalhandler.NewContextWithAuthorization(c)
	if err != nil {
		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	tails, err := strconv.ParseInt(c.Query("tails"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	envName := c.Query("envName")
	productName := c.Query("projectName")

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[productName]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !(ctx.Resources.ProjectAuthInfo[productName].Env.View ||
			ctx.Resources.ProjectAuthInfo[productName].IsProjectAdmin) {
			permitted, err := internalhandler.GetCollaborationModePermission(ctx.UserID, productName, types.ResourceTypeEnvironment, envName, types.EnvActionView)
			if err != nil || !permitted {
				ctx.UnAuthorized = true
				return
			}
		}
	}

	internalhandler.Stream(c, func(ctx context.Context, streamChan chan interface{}) {
		logservice.ContainerLogStream(ctx, streamChan, envName, productName, c.Param("podName"), c.Param("containerName"), true, tails, logger)
	}, logger)
}

func GetProductionEnvContainerLogsSSE(c *gin.Context) {
	logger := ginzap.WithContext(c).Sugar()
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	if err != nil {
		ctx.Err = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	tails, err := strconv.ParseInt(c.Query("tails"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	envName := c.Query("envName")
	productName := c.Query("projectName")

	// authorization checks
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[productName]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !(ctx.Resources.ProjectAuthInfo[productName].ProductionEnv.View ||
			ctx.Resources.ProjectAuthInfo[productName].IsProjectAdmin) {
			permitted, err := internalhandler.GetCollaborationModePermission(ctx.UserID, productName, types.ResourceTypeEnvironment, envName, types.ProductionEnvActionView)
			if err != nil || !permitted {
				ctx.UnAuthorized = true
				return
			}
			ctx.UnAuthorized = true
			return
		}
	}

	internalhandler.Stream(c, func(ctx context.Context, streamChan chan interface{}) {
		logservice.ContainerLogStream(ctx, streamChan, envName, productName, c.Param("podName"), c.Param("containerName"), true, tails, logger)
	}, logger)
}

func GetBuildJobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		internalhandler.JSONResponse(c, ctx)
		return
	}

	tails, err := strconv.ParseInt(c.Param("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}
	subTask := c.Query("subTask")

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.TaskContainerLogStream(
			ctx1, streamChan,
			&logservice.GetContainerOptions{
				Namespace:    config.Namespace(),
				PipelineName: c.Param("pipelineName"),
				SubTask:      subTask,
				TaskID:       taskID,
				TailLines:    tails,
				PipelineType: string(config.SingleType),
			},
			ctx.Logger)
	}, ctx.Logger)
}

func GetWorkflowJobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	taskID, err := strconv.ParseInt(c.Param("taskID"), 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		internalhandler.JSONResponse(c, ctx)
		return
	}

	tails, err := strconv.ParseInt(c.Param("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	jobName := c.Param("jobName")

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.WorkflowTaskV4ContainerLogStream(
			ctx1, streamChan,
			&logservice.GetContainerOptions{
				Namespace:    config.Namespace(),
				PipelineName: c.Param("workflowName"),
				SubTask:      jobcontroller.GetJobContainerName(jobName),
				TaskID:       taskID,
				TailLines:    tails,
			},
			ctx.Logger)
	}, ctx.Logger)
}

func GetWorkflowBuildJobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		internalhandler.JSONResponse(c, ctx)
		return
	}

	tails, err := strconv.ParseInt(c.Param("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	subTask := c.Query("subTask")
	options := &logservice.GetContainerOptions{
		Namespace:     config.Namespace(),
		PipelineName:  c.Param("pipelineName"),
		SubTask:       subTask,
		TailLines:     tails,
		TaskID:        taskID,
		ServiceName:   c.Param("serviceName"),
		ServiceModule: c.Query("serviceModule"),
		PipelineType:  string(config.WorkflowType),
		EnvName:       c.Query("envName"),
		ProductName:   c.Query("projectName"),
	}

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.TaskContainerLogStream(
			ctx1, streamChan,
			options,
			ctx.Logger)
	}, ctx.Logger)
}

func GetTestJobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		internalhandler.JSONResponse(c, ctx)
		return
	}

	tails, err := strconv.ParseInt(c.Param("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	options := &logservice.GetContainerOptions{
		Namespace:    config.Namespace(),
		PipelineName: c.Param("pipelineName"),
		TailLines:    tails,
		TaskID:       taskID,
		PipelineType: string(config.SingleType),
		TestName:     c.Param("testName"),
	}

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.TestJobContainerLogStream(
			ctx1, streamChan,
			options,
			ctx.Logger)
	}, ctx.Logger)
}

func GetWorkflowTestJobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		internalhandler.JSONResponse(c, ctx)
		return
	}

	tails, err := strconv.ParseInt(c.Param("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	workflowTypeString := config.WorkflowType
	workflowType := c.Query("workflowType")
	if workflowType == string(config.TestType) {
		workflowTypeString = config.TestType
	}
	options := &logservice.GetContainerOptions{
		Namespace:    config.Namespace(),
		PipelineName: c.Param("pipelineName"),
		TailLines:    tails,
		TaskID:       taskID,
		PipelineType: string(workflowTypeString),
		ServiceName:  c.Param("serviceName"),
		TestName:     c.Param("testName"),
	}

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.TestJobContainerLogStream(
			ctx1, streamChan,
			options,
			ctx.Logger)
	}, ctx.Logger)
}

func GetServiceJobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)
	defer func() {
		c.Render(-1, sse.Event{
			Event: "job-status",
			Data:  "completed",
		})
	}()

	tails, err := strconv.ParseInt(c.Query("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	subTask := c.Query("subTask")
	options := &logservice.GetContainerOptions{
		Namespace:    config.Namespace(),
		SubTask:      subTask,
		TailLines:    tails,
		ServiceName:  c.Param("serviceName"),
		PipelineType: string(config.ServiceType),
		EnvName:      c.Param("envName"),
		ProductName:  c.Param("productName"),
	}

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.TaskContainerLogStream(
			ctx1, streamChan,
			options,
			ctx.Logger)
	}, ctx.Logger)
}

func GetWorkflowBuildV3JobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		internalhandler.JSONResponse(c, ctx)
		return
	}

	tails, err := strconv.ParseInt(c.Param("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	subTask := c.Query("subTask")
	options := &logservice.GetContainerOptions{
		Namespace:    config.Namespace(),
		PipelineName: c.Param("workflowName"),
		SubTask:      subTask,
		TailLines:    tails,
		TaskID:       taskID,
		PipelineType: string(config.WorkflowTypeV3),
		EnvName:      c.Query("envName"),
		ProductName:  c.Query("projectName"),
		ServiceName:  fmt.Sprintf("%s-job", c.Param("workflowName")),
	}

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.TaskContainerLogStream(
			ctx1, streamChan,
			options,
			ctx.Logger)
	}, ctx.Logger)
}

func GetScanningContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	id := c.Param("id")
	if id == "" {
		ctx.Err = fmt.Errorf("id must be provided")
		return
	}

	taskIDStr := c.Param("scan_id")
	if taskIDStr == "" {
		ctx.Err = fmt.Errorf("scan_id must be provided")
		return
	}

	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		return
	}

	tails, err := strconv.ParseInt(c.Query("lines"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	resp, err := service.GetScanningModuleByID(id, ctx.Logger)

	clusterId := ""
	namespace := config.Namespace()
	if resp.AdvancedSetting != nil {
		clusterId = resp.AdvancedSetting.ClusterID
	}

	scanJobName := fmt.Sprintf("%s-%s", resp.Name, resp.Name)

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.WorkflowTaskV4ContainerLogStream(
			ctx1, streamChan,
			&logservice.GetContainerOptions{
				Namespace:    namespace,
				PipelineName: fmt.Sprintf(setting.ScanWorkflowNamingConvention, id),
				SubTask:      jobcontroller.GetJobContainerName(scanJobName),
				TaskID:       taskID,
				TailLines:    tails,
				ClusterID:    clusterId,
			},
			ctx.Logger)
	}, ctx.Logger)
}

func GetTestingContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	testName := c.Param("test_name")
	if testName == "" {
		ctx.Err = fmt.Errorf("testName must be provided")
		return
	}

	taskIDStr := c.Param("task_id")
	if taskIDStr == "" {
		ctx.Err = fmt.Errorf("task_id must be provided")
		return
	}

	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		return
	}

	tails, err := strconv.ParseInt(c.Query("lines"), 10, 64)
	if err != nil {
		tails = int64(9999999)
	}
	workflowName := fmt.Sprintf(setting.TestWorkflowNamingConvention, testName)
	workflowTask, err := commonrepo.NewworkflowTaskv4Coll().Find(workflowName, taskID)
	if err != nil {
		ctx.Logger.Errorf("failed to find workflow task for testing: %s, err: %s", testName, err)
		ctx.Err = err
		return
	}

	if len(workflowTask.Stages) != 1 {
		ctx.Logger.Errorf("Invalid stage length: stage length for testing should be 1")
		ctx.Err = fmt.Errorf("invalid stage length")
		return
	}

	if len(workflowTask.Stages[0].Jobs) != 1 {
		ctx.Logger.Errorf("Invalid Job length: job length for testing should be 1")
		ctx.Err = fmt.Errorf("invalid job length")
		return
	}

	jobInfo := new(commonmodels.TaskJobInfo)
	if err := commonmodels.IToi(workflowTask.Stages[0].Jobs[0].JobInfo, jobInfo); err != nil {
		ctx.Err = fmt.Errorf("convert job info to task job info error: %v", err)
		return
	}

	buildJobName := strings.ToLower(fmt.Sprintf("%s-%s-%s", jobInfo.JobName, jobInfo.TestingName, jobInfo.RandStr))

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.WorkflowTaskV4ContainerLogStream(
			ctx1, streamChan,
			&logservice.GetContainerOptions{
				Namespace:    config.Namespace(),
				PipelineName: fmt.Sprintf(setting.TestWorkflowNamingConvention, testName),
				SubTask:      jobcontroller.GetJobContainerName(buildJobName),
				TaskID:       taskID,
				TailLines:    tails,
			},
			ctx.Logger)
	}, ctx.Logger)
}

func GetJenkinsJobContainerLogsSSE(c *gin.Context) {
	ctx := internalhandler.NewContext(c)

	jobID, err := strconv.ParseInt(c.Param("jobID"), 10, 64)
	if err != nil {
		ctx.Err = e.ErrInvalidParam.AddDesc("invalid task id")
		internalhandler.JSONResponse(c, ctx)
		return
	}

	internalhandler.Stream(c, func(ctx1 context.Context, streamChan chan interface{}) {
		logservice.JenkinsJobLogStream(ctx1, c.Param("id"), c.Param("jobName"), jobID, streamChan)
	}, ctx.Logger)
}

func OpenAPIGetContainerLogsSSE(c *gin.Context) {
	logger := ginzap.WithContext(c).Sugar()

	tails, err := strconv.ParseInt(c.Query("tails"), 10, 64)
	if err != nil {
		tails = int64(10)
	}

	envName := c.Query("envName")
	productName := c.Query("projectKey")

	internalhandler.Stream(c, func(ctx context.Context, streamChan chan interface{}) {
		logservice.ContainerLogStream(ctx, streamChan, envName, productName, c.Param("podName"), c.Param("containerName"), true, tails, logger)
	}, logger)
}
