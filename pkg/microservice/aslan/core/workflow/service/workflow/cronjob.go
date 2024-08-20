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

package workflow

import (
	"encoding/json"

	"go.uber.org/zap"

	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models/msg_queue"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	commonservice "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service"
	"github.com/koderover/zadig/v2/pkg/setting"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
)

func HandleCronjob(workflow *commonmodels.Workflow, log *zap.SugaredLogger) error {
	workflowSchedule := workflow.Schedules

	if workflowSchedule != nil {
		workflow.Schedules = nil
		workflowSchedule.Enabled = workflow.ScheduleEnabled
		payload := &commonservice.CronjobPayload{
			Name:    workflow.Name,
			JobType: setting.WorkflowCronjob,
		}

		if workflowSchedule.Enabled {
			deleteList, err := UpdateCronjob(workflow.Name, setting.WorkflowCronjob, "", workflowSchedule, log)
			if err != nil {
				log.Errorf("Failed to update cronjob, the error is: %v", err)
				return e.ErrUpsertCronjob.AddDesc(err.Error())
			}
			payload.Action = setting.TypeEnableCronjob
			payload.DeleteList = deleteList
			payload.JobList = workflowSchedule.Items
		} else {
			payload.Action = setting.TypeDisableCronjob
		}

		pl, _ := json.Marshal(payload)
		err := commonrepo.NewMsgQueueCommonColl().Create(&msg_queue.MsgQueueCommon{
			Payload:   string(pl),
			QueueType: setting.TopicCronjob,
		})
		if err != nil {
			log.Errorf("Failed to publish cron to MsgQueueCommon, the error is: %v", err)
			return e.ErrUpsertCronjob.AddDesc(err.Error())
		}
	}
	return nil
}

func UpdateCronjob(parentName, parentType, productName string, schedule *commonmodels.ScheduleCtrl, log *zap.SugaredLogger) (deleteList []string, err error) {
	idMap := make(map[string]bool)
	deleteList = make([]string, 0)
	jobList, err := commonrepo.NewCronjobColl().List(&commonrepo.ListCronjobParam{
		ParentName: parentName,
		ParentType: parentType,
	})

	if err != nil {
		log.Errorf("cannot get cron job list from mongodb, the error is: %v", err)
		return nil, err
	}
	// 把id扔到一个map里面方便统计管理
	for _, cron := range jobList {
		idMap[cron.ID.Hex()] = true
	}
	for _, tasks := range schedule.Items {
		// 非空ID：修改cronjob，保留这个cronjob 空ID: 直接新建条目
		job := &commonmodels.Cronjob{
			Name:         parentName,
			Type:         parentType,
			Number:       tasks.Number,
			Frequency:    tasks.Frequency,
			Time:         tasks.Time,
			Cron:         tasks.Cron,
			MaxFailure:   tasks.MaxFailures,
			TaskArgs:     tasks.TaskArgs,
			WorkflowArgs: tasks.WorkflowArgs,
			TestArgs:     tasks.TestArgs,
			JobType:      string(tasks.Type),
			Enabled:      true,
		}
		if !tasks.ID.IsZero() {
			job.ID = tasks.ID
			if parentType == setting.TestingCronjob {
				job.ProductName = productName
			}
			err := commonrepo.NewCronjobColl().Update(job)
			if err != nil {
				log.Errorf("Failed to update task of id %s, the error is: %v", tasks.ID.Hex(), err)
				return nil, err
			}
			delete(idMap, tasks.ID.Hex())
		} else {
			if parentType == setting.TestingCronjob {
				job.ProductName = productName
			}
			err := commonrepo.NewCronjobColl().Create(job)
			if err != nil {
				log.Errorf("Failed to create task, error: %v", err)
				return nil, err
			}
			tasks.ID = job.ID
		}
	}

	//统计需要删除的cronjob列表
	for k := range idMap {
		deleteList = append(deleteList, k)
	}
	err = commonrepo.NewCronjobColl().Delete(&commonrepo.CronjobDeleteOption{
		IDList: deleteList,
	})
	if err != nil {
		log.Errorf("Failed to delete cronjobs: %v from mongodb, the error is: %v", deleteList, err)
		return nil, err
	}

	return deleteList, nil
}

func DeleteCronjob(parentName, parentType string) error {
	return commonrepo.NewCronjobColl().Delete(&commonrepo.CronjobDeleteOption{
		ParentName: parentName,
		ParentType: parentType,
	})
}
