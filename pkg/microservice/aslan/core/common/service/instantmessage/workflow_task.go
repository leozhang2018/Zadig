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

package instantmessage

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"

	configbase "github.com/koderover/zadig/v2/pkg/config"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service/webhooknotify"
	"github.com/koderover/zadig/v2/pkg/setting"
	userclient "github.com/koderover/zadig/v2/pkg/shared/client/user"
	"github.com/koderover/zadig/v2/pkg/tool/log"
	"github.com/koderover/zadig/v2/pkg/types"
	"github.com/koderover/zadig/v2/pkg/types/step"
)

//go:embed notification.html
var notificationHTML []byte

func (w *Service) SendWorkflowTaskApproveNotifications(workflowName string, taskID int64) error {
	resp, err := w.workflowV4Coll.Find(workflowName)
	if err != nil {
		errMsg := fmt.Sprintf("failed to find workflowv4, err: %s", err)
		log.Error(errMsg)
		return errors.New(errMsg)
	}
	task, err := w.workflowTaskV4Coll.Find(workflowName, taskID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to find workflowv4 task, err: %s", err)
		log.Error(errMsg)
		return errors.New(errMsg)
	}
	for _, notify := range resp.NotifyCtls {
		statusSets := sets.NewString(notify.NotifyTypes...)
		if !statusSets.Has(string(config.StatusWaitingApprove)) {
			continue
		}
		if !notify.Enabled {
			continue
		}
		title, content, larkCard, webhookNotify, err := w.getApproveNotificationContent(notify, task)
		if err != nil {
			errMsg := fmt.Sprintf("failed to get notification content, err: %s", err)
			log.Error(errMsg)
			return errors.New(errMsg)
		}

		if notify.WebHookType == setting.NotifyWebHookTypeMail {
			if task.TaskCreatorID != "" {
				for _, user := range notify.MailUsers {
					if user.Type == setting.UserTypeTaskCreator {
						userInfo, err := userclient.New().GetUserByID(task.TaskCreatorID)
						if err != nil {
							log.Errorf("failed to find user %s, error: %s", task.TaskCreatorID, err)
							break
						}
						notify.MailUsers = append(notify.MailUsers, &models.User{
							Type:     setting.UserTypeUser,
							UserID:   userInfo.Uid,
							UserName: userInfo.Name,
						})
						break
					}
				}
			}
		}

		if err := w.sendNotification(title, content, notify, larkCard, webhookNotify); err != nil {
			log.Errorf("failed to send notification, err: %s", err)
		}
	}
	return nil
}

func (w *Service) SendWorkflowTaskNotifications(task *models.WorkflowTask) error {
	if len(task.OriginWorkflowArgs.NotifyCtls) == 0 {
		return nil
	}
	if task.TaskID <= 0 {
		return nil
	}
	statusChanged := false
	preTask, err := w.workflowTaskV4Coll.Find(task.WorkflowName, task.TaskID-1)
	if err != nil {
		errMsg := fmt.Sprintf("failed to find previous workflowv4, err: %s", err)
		log.Error(errMsg)
		statusChanged = true
	}
	if preTask != nil && task.Status != preTask.Status && task.Status != config.StatusRunning {
		statusChanged = true
	}
	if task.Status == config.StatusCreated {
		statusChanged = false
	}
	for _, notify := range task.OriginWorkflowArgs.NotifyCtls {
		if !notify.Enabled {
			continue
		}
		statusSets := sets.NewString(notify.NotifyTypes...)
		if statusSets.Has(string(task.Status)) || (statusChanged && statusSets.Has(string(config.StatusChanged))) {
			title, content, larkCard, webhookNotify, err := w.getNotificationContent(notify, task)
			if err != nil {
				errMsg := fmt.Sprintf("failed to get notification content, err: %s", err)
				log.Error(errMsg)
				return errors.New(errMsg)
			}

			if notify.WebHookType == setting.NotifyWebHookTypeMail {
				if task.TaskCreatorID != "" {
					for _, user := range notify.MailUsers {
						if user.Type == setting.UserTypeTaskCreator {
							userInfo, err := userclient.New().GetUserByID(task.TaskCreatorID)
							if err != nil {
								log.Errorf("failed to find user %s, error: %s", task.TaskCreatorID, err)
								break
							}
							user.Type = setting.UserTypeUser
							user.UserID = userInfo.Uid
							user.UserName = userInfo.Name
							break
						}
					}
				}
			}

			if err := w.sendNotification(title, content, notify, larkCard, webhookNotify); err != nil {
				log.Errorf("failed to send notification, err: %s", err)
			}
		}
	}
	return nil
}
func (w *Service) getApproveNotificationContent(notify *models.NotifyCtl, task *models.WorkflowTask) (string, string, *LarkCard, *webhooknotify.WorkflowNotify, error) {
	workflowNotification := &workflowTaskNotification{
		Task:               task,
		EncodedDisplayName: url.PathEscape(task.WorkflowDisplayName),
		BaseURI:            configbase.SystemAddress(),
		WebHookType:        notify.WebHookType,
		TotalTime:          time.Now().Unix() - task.StartTime,
	}
	webhookNotify := &webhooknotify.WorkflowNotify{
		TaskID:              task.TaskID,
		WorkflowName:        task.WorkflowName,
		WorkflowDisplayName: task.WorkflowDisplayName,
		ProjectName:         task.ProjectName,
		Status:              task.Status,
		Remark:              task.Remark,
		Error:               task.Error,
		CreateTime:          task.CreateTime,
		StartTime:           task.StartTime,
		EndTime:             task.EndTime,
		TaskCreator:         task.TaskCreator,
		TaskCreatorID:       task.TaskCreatorID,
		TaskCreatorPhone:    task.TaskCreatorPhone,
		TaskCreatorEmail:    task.TaskCreatorEmail,
	}

	tplTitle := "{{if ne .WebHookType \"feishu\"}}#### {{end}}{{getIcon .Task.Status }}{{if eq .WebHookType \"wechat\"}}<font color=\"markdownColorInfo\">工作流{{.Task.WorkflowDisplayName}} #{{.Task.TaskID}} 等待审批</font>{{else}}工作流 {{.Task.WorkflowDisplayName}} #{{.Task.TaskID}} 等待审批{{end}} \n"
	mailTplTitle := "{{getIcon .Task.Status }}工作流 {{.Task.WorkflowDisplayName}} #{{.Task.TaskID}} 等待审批\n"

	tplBaseInfo := []string{"{{if eq .WebHookType \"dingding\"}}##### {{end}}**执行用户**：{{.Task.TaskCreator}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**项目名称**：{{.Task.ProjectName}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**开始时间**：{{ getStartTime .Task.StartTime}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**持续时间**：{{ getDuration .TotalTime}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**备注**：{{.Task.Remark}} \n",
	}
	mailTplBaseInfo := []string{"执行用户：{{.Task.TaskCreator}} \n",
		"项目名称：{{.Task.ProjectName}} \n",
		"开始时间：{{ getStartTime .Task.StartTime}} \n",
		"持续时间：{{ getDuration .TotalTime}} \n",
		"备注：{{ .Task.Remark}} \n\n",
	}

	title, err := getWorkflowTaskTplExec(tplTitle, workflowNotification)
	if err != nil {
		return "", "", nil, nil, err
	}

	buttonContent := "点击查看更多信息"
	workflowDetailURL := "{{.BaseURI}}/v1/projects/detail/{{.Task.ProjectName}}/pipelines/custom/{{.Task.WorkflowName}}/{{.Task.TaskID}}?display_name={{.EncodedDisplayName}}"
	moreInformation := fmt.Sprintf("[%s](%s)", buttonContent, workflowDetailURL)
	if notify.WebHookType == setting.NotifyWebHookTypeMail {
		title, err = getWorkflowTaskTplExec(mailTplTitle, workflowNotification)
		if err != nil {
			return "", "", nil, nil, err
		}

		tplcontent := strings.Join(mailTplBaseInfo, "")
		content, err := getWorkflowTaskTplExec(tplcontent, workflowNotification)
		if err != nil {
			return "", "", nil, nil, err
		}
		content = strings.TrimSpace(content)

		t, err := template.New("workflow_notification").Parse(string(notificationHTML))
		if err != nil {
			err = fmt.Errorf("workflow notification template parse error, error msg:%s", err)
			return "", "", nil, nil, err
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, struct {
			WorkflowName   string
			WorkflowTaskID int64
			Content        string
			Url            string
		}{
			WorkflowName:   task.WorkflowDisplayName,
			WorkflowTaskID: task.TaskID,
			Content:        content,
			Url:            fmt.Sprintf("%s/v1/projects/detail/%s/pipelines/custom/%s?display_name=%s", configbase.SystemAddress(), task.ProjectName, task.WorkflowName, url.PathEscape(task.WorkflowDisplayName)),
		})
		if err != nil {
			err = fmt.Errorf("workflow notification template execute error, error msg:%s", err)
			return "", "", nil, nil, err
		}

		content = buf.String()
		return title, content, nil, nil, nil
	} else if notify.WebHookType == setting.NotifyWebHookTypeWebook {
		webhookNotify.DetailURL = fmt.Sprintf("%s/v1/projects/detail/%s/pipelines/custom/%s?display_name=%s", configbase.SystemAddress(), task.ProjectName, task.WorkflowName, url.PathEscape(task.WorkflowDisplayName))
		return "", "", nil, webhookNotify, nil
	} else if notify.WebHookType != setting.NotifyWebHookTypeFeishu {
		tplcontent := strings.Join(tplBaseInfo, "")
		tplcontent = tplcontent + getNotifyAtContent(notify)
		tplcontent = fmt.Sprintf("%s%s%s", title, tplcontent, moreInformation)
		content, err := getWorkflowTaskTplExec(tplcontent, workflowNotification)
		if err != nil {
			return "", "", nil, nil, err
		}
		return title, content, nil, nil, nil
	}

	lc := NewLarkCard()
	lc.SetConfig(true)
	lc.SetHeader(feishuHeaderTemplateGreen, title, feiShuTagText)
	for idx, feildContent := range tplBaseInfo {
		feildExecContent, _ := getWorkflowTaskTplExec(feildContent, workflowNotification)
		lc.AddI18NElementsZhcnFeild(feildExecContent, idx == 0)
	}
	workflowDetailURL, _ = getWorkflowTaskTplExec(workflowDetailURL, workflowNotification)
	lc.AddI18NElementsZhcnAction(buttonContent, workflowDetailURL)
	return "", "", lc, nil, nil
}

// @note custom workflow task v4 notification
func (w *Service) getNotificationContent(notify *models.NotifyCtl, task *models.WorkflowTask) (string, string, *LarkCard, *webhooknotify.WorkflowNotify, error) {
	workflowNotification := &workflowTaskNotification{
		Task:               task,
		EncodedDisplayName: url.PathEscape(task.WorkflowDisplayName),
		BaseURI:            configbase.SystemAddress(),
		WebHookType:        notify.WebHookType,
		TotalTime:          time.Now().Unix() - task.StartTime,
	}

	if task.Type == config.WorkflowTaskTypeScanning {
		segs := strings.Split(task.WorkflowName, "-")
		workflowNotification.ScanningID = segs[len(segs)-1]
	}

	webhookNotify := &webhooknotify.WorkflowNotify{
		TaskID:              task.TaskID,
		WorkflowName:        task.WorkflowName,
		WorkflowDisplayName: task.WorkflowDisplayName,
		ProjectName:         task.ProjectName,
		Status:              task.Status,
		Remark:              task.Remark,
		Error:               task.Error,
		CreateTime:          task.CreateTime,
		StartTime:           task.StartTime,
		EndTime:             task.EndTime,
		TaskCreator:         task.TaskCreator,
		TaskCreatorID:       task.TaskCreatorID,
		TaskCreatorPhone:    task.TaskCreatorPhone,
		TaskCreatorEmail:    task.TaskCreatorEmail,
	}

	tplTitle := "{{if ne .WebHookType \"feishu\"}}#### {{end}}{{getIcon .Task.Status }}{{if eq .WebHookType \"wechat\"}}<font color=\"{{ getColor .Task.Status }}\">工作流{{.Task.WorkflowDisplayName}} #{{.Task.TaskID}} {{ taskStatus .Task.Status }}</font>{{else}}工作流 {{.Task.WorkflowDisplayName}} #{{.Task.TaskID}} {{ taskStatus .Task.Status }}{{end}} \n"
	mailTplTitle := "{{getIcon .Task.Status }} 工作流 {{.Task.WorkflowDisplayName}}#{{.Task.TaskID}} {{ taskStatus .Task.Status }}"

	tplBaseInfo := []string{"{{if eq .WebHookType \"dingding\"}}##### {{end}}**执行用户**：{{.Task.TaskCreator}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**项目名称**：{{.Task.ProjectName}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**开始时间**：{{ getStartTime .Task.StartTime}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**持续时间**：{{ getDuration .TotalTime}} \n",
		"{{if eq .WebHookType \"dingding\"}}##### {{end}}**备注**：{{.Task.Remark}} \n",
	}
	mailTplBaseInfo := []string{"执行用户：{{.Task.TaskCreator}} \n",
		"项目名称：{{.Task.ProjectName}} \n",
		"开始时间：{{ getStartTime .Task.StartTime}} \n",
		"持续时间：{{ getDuration .TotalTime}} \n",
		"备注：{{ .Task.Remark}} \n",
	}

	jobContents := []string{}
	workflowNotifyStages := []*webhooknotify.WorkflowNotifyStage{}
	for _, stage := range task.Stages {
		workflowNotifyStage := &webhooknotify.WorkflowNotifyStage{
			Name:      stage.Name,
			Status:    stage.Status,
			StartTime: stage.StartTime,
			EndTime:   stage.EndTime,
			Error:     stage.Error,
		}

		for _, job := range stage.Jobs {
			workflowNotifyJob := &webhooknotify.WorkflowNotifyJobTask{
				Name:      job.Name,
				JobType:   job.JobType,
				Status:    job.Status,
				StartTime: job.StartTime,
				EndTime:   job.EndTime,
				Error:     job.Error,
			}

			jobTplcontent := "{{if ne .WebHookType \"feishu\"}}\n\n{{end}}{{if eq .WebHookType \"dingding\"}}---\n\n##### {{end}}**{{jobType .Job.JobType }}**: {{.Job.Name}}    **状态**: {{taskStatus .Job.Status }} \n"
			mailJobTplcontent := "{{jobType .Job.JobType }}：{{.Job.Name}}    状态：{{taskStatus .Job.Status }} \n"
			switch job.JobType {
			case string(config.JobZadigBuild):
				fallthrough
			case string(config.JobFreestyle):
				jobSpec := &models.JobTaskFreestyleSpec{}
				models.IToi(job.Spec, jobSpec)

				workflowNotifyJobTaskSpec := &webhooknotify.WorkflowNotifyJobTaskBuildSpec{}

				repos := []*types.Repository{}
				for _, stepTask := range jobSpec.Steps {
					if stepTask.StepType == config.StepGit {
						stepSpec := &step.StepGitSpec{}
						models.IToi(stepTask.Spec, stepSpec)
						repos = stepSpec.Repos
					}
				}
				branchTag, commitID, gitCommitURL := "", "", ""
				commitMsgs := []string{}
				var prInfoList []string
				var prInfo string
				for idx, buildRepo := range repos {
					workflowNotifyRepository := &webhooknotify.WorkflowNotifyRepository{
						Source:        buildRepo.Source,
						RepoOwner:     buildRepo.RepoOwner,
						RepoNamespace: buildRepo.RepoNamespace,
						RepoName:      buildRepo.RepoName,
						Branch:        buildRepo.Branch,
						Tag:           buildRepo.Tag,
						CommitID:      buildRepo.CommitID,
						CommitMessage: buildRepo.CommitMessage,
					}
					if idx == 0 || buildRepo.IsPrimary {
						branchTag = buildRepo.Branch
						if buildRepo.Tag != "" {
							branchTag = buildRepo.Tag
						}
						if len(buildRepo.CommitID) > 8 {
							commitID = buildRepo.CommitID[0:8]
						}
						var prLinkBuilder func(baseURL, owner, repoName string, prID int) string
						switch buildRepo.Source {
						case types.ProviderGithub:
							prLinkBuilder = func(baseURL, owner, repoName string, prID int) string {
								return fmt.Sprintf("%s/%s/%s/pull/%d", baseURL, owner, repoName, prID)
							}
						case types.ProviderGitee:
							prLinkBuilder = func(baseURL, owner, repoName string, prID int) string {
								return fmt.Sprintf("%s/%s/%s/pulls/%d", baseURL, owner, repoName, prID)
							}
						case types.ProviderGitlab:
							prLinkBuilder = func(baseURL, owner, repoName string, prID int) string {
								return fmt.Sprintf("%s/%s/%s/merge_requests/%d", baseURL, owner, repoName, prID)
							}
						case types.ProviderGerrit:
							prLinkBuilder = func(baseURL, owner, repoName string, prID int) string {
								return fmt.Sprintf("%s/%d", baseURL, prID)
							}
						default:
							prLinkBuilder = func(baseURL, owner, repoName string, prID int) string {
								return ""
							}
						}
						prInfoList = []string{}
						sort.Ints(buildRepo.PRs)
						for _, id := range buildRepo.PRs {
							link := prLinkBuilder(buildRepo.Address, buildRepo.RepoOwner, buildRepo.RepoName, id)
							if link != "" {
								prInfoList = append(prInfoList, fmt.Sprintf("[#%d](%s)", id, link))
							}
						}
						commitMsg := strings.Trim(buildRepo.CommitMessage, "\n")
						commitMsgs = strings.Split(commitMsg, "\n")
						gitCommitURL = fmt.Sprintf("%s/%s/%s/commit/%s", buildRepo.Address, buildRepo.RepoOwner, buildRepo.RepoName, commitID)
						workflowNotifyRepository.CommitURL = gitCommitURL
					}

					workflowNotifyJobTaskSpec.Repositories = append(workflowNotifyJobTaskSpec.Repositories, workflowNotifyRepository)
				}
				if len(prInfoList) != 0 {
					// need an extra space at the end
					prInfo = strings.Join(prInfoList, " ") + " "
				}
				image := ""
				for _, env := range jobSpec.Properties.Envs {
					if env.Key == "IMAGE" {
						image = env.Value
					}
				}
				if len(commitID) > 0 {
					jobTplcontent += fmt.Sprintf("{{if eq .WebHookType \"dingding\"}}##### {{end}}**代码信息**：%s %s[%s](%s) \n", branchTag, prInfo, commitID, gitCommitURL)
					jobTplcontent += "{{if eq .WebHookType \"dingding\"}}##### {{end}}**提交信息**："
					mailJobTplcontent += fmt.Sprintf("代码信息：%s %s[%s]( %s )\n", branchTag, prInfo, commitID, gitCommitURL)
					if len(commitMsgs) == 1 {
						jobTplcontent += fmt.Sprintf("%s \n", commitMsgs[0])
					} else {
						jobTplcontent += "\n"
						for _, commitMsg := range commitMsgs {
							jobTplcontent += fmt.Sprintf("%s \n", commitMsg)
						}
					}
				}
				if image != "" {
					jobTplcontent += fmt.Sprintf("{{if eq .WebHookType \"dingding\"}}##### {{end}}**镜像信息**：%s \n", image)
					mailJobTplcontent += fmt.Sprintf("镜像信息：%s \n", image)
					workflowNotifyJobTaskSpec.Image = image
				}

				workflowNotifyJob.Spec = workflowNotifyJobTaskSpec
			case string(config.JobZadigDeploy):
				jobSpec := &models.JobTaskDeploySpec{}
				models.IToi(job.Spec, jobSpec)
				jobTplcontent += fmt.Sprintf("{{if eq .WebHookType \"dingding\"}}##### {{end}}**环境**：%s \n", jobSpec.Env)
				mailJobTplcontent += fmt.Sprintf("环境：%s \n", jobSpec.Env)

				serviceModules := []*webhooknotify.WorkflowNotifyDeployServiceModule{}
				for _, serviceAndImage := range jobSpec.ServiceAndImages {
					serviceModule := &webhooknotify.WorkflowNotifyDeployServiceModule{
						ServiceModule: serviceAndImage.ServiceModule,
						Image:         serviceAndImage.Image,
					}
					serviceModules = append(serviceModules, serviceModule)
				}

				workflowNotifyJobTaskSpec := &webhooknotify.WorkflowNotifyJobTaskDeploySpec{
					Env:            jobSpec.Env,
					ServiceName:    jobSpec.ServiceName,
					ServiceModules: serviceModules,
				}
				workflowNotifyJob.Spec = workflowNotifyJobTaskSpec
			case string(config.JobZadigHelmDeploy):
				jobSpec := &models.JobTaskHelmDeploySpec{}
				models.IToi(job.Spec, jobSpec)
				jobTplcontent += fmt.Sprintf("{{if eq .WebHookType \"dingding\"}}##### {{end}}**环境**：%s \n", jobSpec.Env)
				mailJobTplcontent += fmt.Sprintf("环境：%s \n", jobSpec.Env)

				serviceModules := []*webhooknotify.WorkflowNotifyDeployServiceModule{}
				for _, serviceAndImage := range jobSpec.ImageAndModules {
					serviceModule := &webhooknotify.WorkflowNotifyDeployServiceModule{
						ServiceModule: serviceAndImage.ServiceModule,
						Image:         serviceAndImage.Image,
					}
					serviceModules = append(serviceModules, serviceModule)
				}

				workflowNotifyJobTaskSpec := &webhooknotify.WorkflowNotifyJobTaskDeploySpec{
					Env:            jobSpec.Env,
					ServiceName:    jobSpec.ServiceName,
					ServiceModules: serviceModules,
				}
				workflowNotifyJob.Spec = workflowNotifyJobTaskSpec
			}
			jobNotifaication := &jobTaskNotification{
				Job:         job,
				WebHookType: notify.WebHookType,
			}

			if notify.WebHookType == setting.NotifyWebHookTypeMail {
				jobContent, err := getJobTaskTplExec(mailJobTplcontent, jobNotifaication)
				if err != nil {
					return "", "", nil, nil, err
				}
				jobContents = append(jobContents, jobContent)
			} else {
				jobContent, err := getJobTaskTplExec(jobTplcontent, jobNotifaication)
				if err != nil {
					return "", "", nil, nil, err
				}
				jobContents = append(jobContents, jobContent)
			}

			workflowNotifyStage.Jobs = append(workflowNotifyStage.Jobs, workflowNotifyJob)
		}
		workflowNotifyStages = append(workflowNotifyStages, workflowNotifyStage)
	}
	webhookNotify.Stages = workflowNotifyStages

	title, err := getWorkflowTaskTplExec(tplTitle, workflowNotification)
	if err != nil {
		return "", "", nil, nil, err
	}
	buttonContent := "点击查看更多信息"
	workflowDetailURL := ""
	switch task.Type {
	case config.WorkflowTaskTypeWorkflow:
		workflowDetailURL = "{{.BaseURI}}/v1/projects/detail/{{.Task.ProjectName}}/pipelines/custom/{{.Task.WorkflowName}}/{{.Task.TaskID}}?display_name={{.EncodedDisplayName}}"
	case config.WorkflowTaskTypeScanning:
		workflowDetailURL = "{{.BaseURI}}/v1/projects/detail/{{.Task.ProjectName}}/scanner/detail/{{.Task.WorkflowDisplayName}}/task/{{.Task.TaskID}}?status={{.Task.Status}}&id={{.ScanningID}}"
	case config.WorkflowTaskTypeTesting:
		workflowDetailURL = "{{.BaseURI}}/v1/projects/detail/{{.Task.ProjectName}}/test/detail/function/{{.Task.WorkflowDisplayName}}/{{.Task.TaskID}}?status={{.Task.Status}}&id=&display_name={{.Task.WorkflowDisplayName}}"
	default:
		workflowDetailURL = "{{.BaseURI}}/v1/projects/detail/{{.Task.ProjectName}}/pipelines/custom/{{.Task.WorkflowName}}/{{.Task.TaskID}}?display_name={{.EncodedDisplayName}}"
	}
	moreInformation := fmt.Sprintf("\n\n{{if eq .WebHookType \"dingding\"}}---\n\n{{end}}[%s](%s)", buttonContent, workflowDetailURL)

	if notify.WebHookType == setting.NotifyWebHookTypeMail {
		title, err := getWorkflowTaskTplExec(mailTplTitle, workflowNotification)
		if err != nil {
			return "", "", nil, nil, err
		}

		tplcontent := strings.Join(mailTplBaseInfo, "")
		tplcontent += strings.Join(jobContents, "")
		content, err := getWorkflowTaskTplExec(tplcontent, workflowNotification)
		if err != nil {
			return "", "", nil, nil, err
		}
		content = strings.TrimSpace(content)

		t, err := template.New("workflow_notification").Parse(string(notificationHTML))
		if err != nil {
			err = fmt.Errorf("workflow notification template parse error, error msg:%s", err)
			return "", "", nil, nil, err
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, struct {
			WorkflowName   string
			WorkflowTaskID int64
			Content        string
			Url            string
		}{
			WorkflowName:   task.WorkflowDisplayName,
			WorkflowTaskID: task.TaskID,
			Content:        content,
			Url:            fmt.Sprintf("%s/v1/projects/detail/%s/pipelines/custom/%s?display_name=%s", configbase.SystemAddress(), task.ProjectName, task.WorkflowName, url.PathEscape(task.WorkflowDisplayName)),
		})
		if err != nil {
			err = fmt.Errorf("workflow notification template execute error, error msg:%s", err)
			return "", "", nil, nil, err
		}

		content = buf.String()
		return title, content, nil, nil, nil
	} else if notify.WebHookType == setting.NotifyWebHookTypeWebook {
		webhookNotify.DetailURL = fmt.Sprintf("%s/v1/projects/detail/%s/pipelines/custom/%s?display_name=%s", configbase.SystemAddress(), task.ProjectName, task.WorkflowName, url.PathEscape(task.WorkflowDisplayName))
		return "", "", nil, webhookNotify, nil
	} else if notify.WebHookType != setting.NotifyWebHookTypeFeishu {
		tplcontent := strings.Join(tplBaseInfo, "")
		tplcontent += strings.Join(jobContents, "")
		tplcontent = tplcontent + getNotifyAtContent(notify)
		tplcontent = fmt.Sprintf("%s%s%s", title, tplcontent, moreInformation)
		content, err := getWorkflowTaskTplExec(tplcontent, workflowNotification)
		if err != nil {
			return "", "", nil, nil, err
		}

		return title, content, nil, nil, nil
	}

	lc := NewLarkCard()
	lc.SetConfig(true)
	lc.SetHeader(getColorTemplateWithStatus(task.Status), title, feiShuTagText)
	for idx, feildContent := range tplBaseInfo {
		feildExecContent, _ := getWorkflowTaskTplExec(feildContent, workflowNotification)
		lc.AddI18NElementsZhcnFeild(feildExecContent, idx == 0)
	}
	for _, feildContent := range jobContents {
		feildExecContent, _ := getWorkflowTaskTplExec(feildContent, workflowNotification)
		lc.AddI18NElementsZhcnFeild(feildExecContent, true)
	}
	workflowDetailURL, _ = getWorkflowTaskTplExec(workflowDetailURL, workflowNotification)
	lc.AddI18NElementsZhcnAction(buttonContent, workflowDetailURL)
	return "", "", lc, nil, nil
}

type workflowTaskNotification struct {
	Task               *models.WorkflowTask      `json:"task"`
	EncodedDisplayName string                    `json:"encoded_display_name"`
	BaseURI            string                    `json:"base_uri"`
	WebHookType        setting.NotifyWebHookType `json:"web_hook_type"`
	TotalTime          int64                     `json:"total_time"`
	ScanningID         string                    `json:"scanning_id"`
}

func getWorkflowTaskTplExec(tplcontent string, args *workflowTaskNotification) (string, error) {
	tmpl := template.Must(template.New("notify").Funcs(template.FuncMap{
		"getColor": func(status config.Status) string {
			if status == config.StatusPassed || status == config.StatusCreated {
				return markdownColorInfo
			} else if status == config.StatusTimeout || status == config.StatusCancelled {
				return markdownColorComment
			} else if status == config.StatusFailed {
				return markdownColorWarning
			}
			return markdownColorComment
		},
		"taskStatus": func(status config.Status) string {
			if status == config.StatusPassed {
				return "执行成功"
			} else if status == config.StatusCancelled {
				return "执行取消"
			} else if status == config.StatusTimeout {
				return "执行超时"
			} else if status == config.StatusReject {
				return "执行被拒绝"
			} else if status == config.StatusCreated {
				return "开始执行"
			}
			return "执行失败"
		},
		"getIcon": func(status config.Status) string {
			if status == config.StatusPassed || status == config.StatusCreated {
				return "👍"
			} else if status == config.StatusFailed {
				return "❌"
			}
			return "⚠️"
		},
		"getStartTime": func(startTime int64) string {
			return time.Unix(startTime, 0).Format("2006-01-02 15:04:05")
		},
		"getDuration": func(startTime int64) string {
			duration, er := time.ParseDuration(strconv.FormatInt(startTime, 10) + "s")
			if er != nil {
				log.Errorf("getTplExec ParseDuration err:%s", er)
				return "0s"
			}
			return duration.String()
		},
	}).Parse(tplcontent))

	buffer := bytes.NewBufferString("")
	if err := tmpl.Execute(buffer, args); err != nil {
		log.Errorf("getTplExec Execute err:%s", err)
		return "", fmt.Errorf("getTplExec Execute err:%s", err)

	}
	return buffer.String(), nil
}

type jobTaskNotification struct {
	Job         *models.JobTask           `json:"task"`
	WebHookType setting.NotifyWebHookType `json:"web_hook_type"`
}

func getJobTaskTplExec(tplcontent string, args *jobTaskNotification) (string, error) {
	tmpl := template.Must(template.New("notify").Funcs(template.FuncMap{
		"taskStatus": func(status config.Status) string {
			if status == config.StatusPassed {
				return "执行成功"
			} else if status == config.StatusCancelled {
				return "执行取消"
			} else if status == config.StatusTimeout {
				return "执行超时"
			} else if status == config.StatusReject {
				return "执行被拒绝"
			} else if status == "" {
				return "未执行"
			}
			return "执行失败"
		},
		"jobType": func(jobType string) string {
			switch jobType {
			case string(config.JobZadigBuild):
				return "构建"
			case string(config.JobZadigDeploy):
				return "部署"
			case string(config.JobZadigHelmDeploy):
				return "helm部署"
			case string(config.JobCustomDeploy):
				return "自定义部署"
			case string(config.JobFreestyle):
				return "通用任务"
			case string(config.JobPlugin):
				return "自定义任务"
			case string(config.JobZadigTesting):
				return "测试"
			case string(config.JobZadigScanning):
				return "代码扫描"
			case string(config.JobZadigDistributeImage):
				return "镜像分发"
			case string(config.JobK8sBlueGreenDeploy):
				return "蓝绿部署"
			case string(config.JobK8sBlueGreenRelease):
				return "蓝绿发布"
			case string(config.JobK8sCanaryDeploy):
				return "金丝雀部署"
			case string(config.JobK8sCanaryRelease):
				return "金丝雀发布"
			case string(config.JobK8sGrayRelease):
				return "灰度发布"
			case string(config.JobK8sGrayRollback):
				return "灰度回滚"
			case string(config.JobK8sPatch):
				return "更新 k8s YAML"
			case string(config.JobIstioRelease):
				return "istio 发布"
			case string(config.JobIstioRollback):
				return "istio 回滚"
			case string(config.JobJira):
				return "jira 问题状态变更"
			case string(config.JobNacos):
				return "Nacos 配置变更"
			case string(config.JobApollo):
				return "Apollo 配置变更"
			case string(config.JobMeegoTransition):
				return "飞书工作项状态变更"
			default:
				return string(jobType)
			}
		},
	}).Parse(tplcontent))

	buffer := bytes.NewBufferString("")
	if err := tmpl.Execute(buffer, args); err != nil {
		log.Errorf("getTplExec Execute err:%s", err)
		return "", fmt.Errorf("getTplExec Execute err:%s", err)

	}
	return buffer.String(), nil
}

func (w *Service) sendNotification(title, content string, notify *models.NotifyCtl, card *LarkCard, webhookNotify *webhooknotify.WorkflowNotify) error {
	switch notify.WebHookType {
	case setting.NotifyWebHookTypeDingDing:
		if err := w.sendDingDingMessage(notify.DingDingWebHook, title, content, notify.AtMobiles, notify.IsAtAll); err != nil {
			return err
		}
	case setting.NotifyWebHookTypeFeishu:
		if err := w.sendFeishuMessage(notify.FeiShuWebHook, card); err != nil {
			return err
		}
		if err := w.sendFeishuMessageOfSingleType("", notify.FeiShuWebHook, getNotifyAtContent(notify)); err != nil {
			return err
		}
	case setting.NotifyWebHookTypeMail:
		if err := w.sendMailMessage(title, content, notify.MailUsers); err != nil {
			return err
		}
	case setting.NotifyWebHookTypeWebook:
		webhookclient := webhooknotify.NewClient(notify.WebHookNotify.Address, notify.WebHookNotify.Token)
		err := webhookclient.SendWorkflowWebhook(webhookNotify)
		if err != nil {
			return fmt.Errorf("failed to send notification to webhook, address %s, token: %s, error: %v", notify.WebHookNotify.Address, notify.WebHookNotify.Token, err)
		}
	default:
		if err := w.SendWeChatWorkMessage(weChatTextTypeMarkdown, notify.WeChatWebHook, content); err != nil {
			return err
		}
	}
	return nil
}
