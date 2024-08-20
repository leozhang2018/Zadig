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

package job

import (
	"context"
	"fmt"
	"strings"

	"github.com/koderover/zadig/v2/pkg/setting"
	"github.com/koderover/zadig/v2/pkg/tool/sonar"
	"github.com/koderover/zadig/v2/pkg/types/step"
	"go.uber.org/zap"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	commonservice "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service"
	"github.com/koderover/zadig/v2/pkg/tool/log"
	"github.com/koderover/zadig/v2/pkg/types"
)

type ScanningJob struct {
	job      *commonmodels.Job
	workflow *commonmodels.WorkflowV4
	spec     *commonmodels.ZadigScanningJobSpec
}

func (j *ScanningJob) Instantiate() error {
	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToiYaml(j.job.Spec, j.spec); err != nil {
		return err
	}
	j.job.Spec = j.spec
	return nil
}

func (j *ScanningJob) SetPreset() error {
	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}

	if j.spec.ScanningType == config.NormalScanningType {
		for _, scanning := range j.spec.Scannings {
			scanningInfo, err := commonrepo.NewScanningColl().Find(j.workflow.Project, scanning.Name)
			if err != nil {
				log.Errorf("find scanning: %s error: %v", scanning.Name, err)
				continue
			}
			if err := fillScanningDetail(scanningInfo); err != nil {
				log.Errorf("fill scanning: %s detail error: %v", scanningInfo.Name, err)
				continue
			}
			scanning.Repos = mergeRepos(scanningInfo.Repos, scanning.Repos)
			scanning.KeyVals = renderKeyVals(scanning.KeyVals, scanningInfo.Envs)
		}
	}

	// if quoted job quote another job, then use the service and image of the quoted job
	if j.spec.Source == config.SourceFromJob {
		j.spec.OriginJobName = j.spec.JobName
	}

	if j.spec.ScanningType == config.ServiceScanningType {
		serviceMap := map[string]bool{}
		for _, scanning := range j.spec.ServiceAndScannings {
			serviceMap[fmt.Sprintf("%s/%s", scanning.ServiceName, scanning.ServiceModule)] = true
			scanningInfo, err := commonrepo.NewScanningColl().Find(j.workflow.Project, scanning.Name)
			if err != nil {
				log.Errorf("find testing: %s error: %v", scanning.Name, err)
				continue
			}
			if err := fillScanningDetail(scanningInfo); err != nil {
				log.Errorf("fill scanning: %s detail error: %v", scanningInfo.Name, err)
				continue
			}
			scanning.Repos = mergeRepos(scanningInfo.Repos, scanning.Repos)
			scanning.KeyVals = renderKeyVals(scanning.KeyVals, scanningInfo.Envs)
		}
		filteredTargets := []*commonmodels.ServiceTestTarget{}
		for _, target := range j.spec.TargetServices {
			if _, ok := serviceMap[fmt.Sprintf("%s/%s", target.ServiceName, target.ServiceModule)]; ok {
				filteredTargets = append(filteredTargets, target)
			}
		}
		j.spec.TargetServices = filteredTargets
	}

	j.job.Spec = j.spec
	return nil
}

func (j *ScanningJob) SetOptions() error {
	return nil
}

func (j *ScanningJob) ClearSelectionField() error {
	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToiYaml(j.job.Spec, j.spec); err != nil {
		return err
	}
	j.spec.TargetServices = make([]*commonmodels.ServiceTestTarget, 0)
	j.job.Spec = j.spec
	return nil
}

func (j *ScanningJob) UpdateWithLatestSetting() error {
	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}

	latestWorkflow, err := commonrepo.NewWorkflowV4Coll().Find(j.workflow.Name)
	if err != nil {
		log.Errorf("Failed to find original workflow to set options, error: %s", err)
	}

	latestSpec := new(commonmodels.ZadigScanningJobSpec)
	found := false
	for _, stage := range latestWorkflow.Stages {
		if !found {
			for _, job := range stage.Jobs {
				if job.Name == j.job.Name && job.JobType == j.job.JobType {
					if err := commonmodels.IToi(job.Spec, latestSpec); err != nil {
						return err
					}
					found = true
					break
				}
			}
		} else {
			break
		}
	}

	if !found {
		return fmt.Errorf("failed to find the original workflow: %s", j.workflow.Name)
	}

	mergedScanning := make([]*commonmodels.ScanningModule, 0)

	for _, latestScanning := range latestSpec.Scannings {
		for _, scanning := range j.spec.Scannings {
			if scanning.Name == latestScanning.Name && scanning.ProjectName == latestScanning.ProjectName {
				scanningInfo, err := commonrepo.NewScanningColl().Find(j.workflow.Project, latestScanning.Name)
				if err != nil {
					log.Errorf("find scanning: %s error: %v", scanning.Name, err)
					continue
				}
				if err := fillScanningDetail(scanningInfo); err != nil {
					log.Errorf("fill scanning: %s detail error: %v", scanningInfo.Name, err)
					continue
				}

				latestScanning.Repos = mergeRepos(scanningInfo.Repos, latestScanning.Repos)
				latestScanning.KeyVals = renderKeyVals(latestScanning.KeyVals, scanningInfo.Envs)
				scanning.Repos = mergeRepos(latestScanning.Repos, scanning.Repos)
				scanning.KeyVals = renderKeyVals(scanning.KeyVals, latestScanning.KeyVals)

				mergedScanning = append(mergedScanning, scanning)
			}
		}
	}

	j.spec.Scannings = mergedScanning
	j.job.Spec = j.spec
	return nil
}

func (j *ScanningJob) GetRepos() ([]*types.Repository, error) {
	resp := []*types.Repository{}
	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return resp, err
	}

	for _, scanning := range j.spec.Scannings {
		scanningInfo, err := commonrepo.NewScanningColl().Find(j.workflow.Project, scanning.Name)
		if err != nil {
			log.Errorf("find scanning: %s error: %v", scanning.Name, err)
			continue
		}
		resp = append(resp, mergeRepos(scanningInfo.Repos, scanning.Repos)...)
	}
	return resp, nil
}

func (j *ScanningJob) MergeArgs(args *commonmodels.Job) error {
	if j.job.Name == args.Name && j.job.JobType == args.JobType {
		j.spec = &commonmodels.ZadigScanningJobSpec{}
		if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
			return err
		}
		j.job.Spec = j.spec
		argsSpec := &commonmodels.ZadigScanningJobSpec{}
		if err := commonmodels.IToi(args.Spec, argsSpec); err != nil {
			return err
		}

		if j.spec.ScanningType == config.NormalScanningType {
			for _, scanning := range j.spec.Scannings {
				for _, argsScanning := range argsSpec.Scannings {
					if scanning.Name == argsScanning.Name {
						scanning.Repos = mergeRepos(scanning.Repos, argsScanning.Repos)
						scanning.KeyVals = renderKeyVals(argsScanning.KeyVals, scanning.KeyVals)
						break
					}
				}
			}
		}

		if j.spec.ScanningType == config.ServiceScanningType {
			j.spec.TargetServices = argsSpec.TargetServices
			for _, testing := range j.spec.ServiceAndScannings {
				for _, argsTesting := range argsSpec.ServiceAndScannings {
					if testing.Name == argsTesting.Name && testing.ServiceName == argsTesting.ServiceName {
						testing.Repos = mergeRepos(testing.Repos, argsTesting.Repos)
						testing.KeyVals = renderKeyVals(argsTesting.KeyVals, testing.KeyVals)
						break
					}
				}
			}
		}

		j.job.Spec = j.spec
	}
	return nil
}

func (j *ScanningJob) MergeWebhookRepo(webhookRepo *types.Repository) error {
	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}
	for _, scanning := range j.spec.Scannings {
		scanning.Repos = mergeRepos(scanning.Repos, []*types.Repository{webhookRepo})
	}
	j.job.Spec = j.spec
	return nil
}

func (j *ScanningJob) ToJobs(taskID int64) ([]*commonmodels.JobTask, error) {
	logger := log.SugaredLogger()
	resp := []*commonmodels.JobTask{}

	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return resp, err
	}

	if j.spec.ScanningType == config.NormalScanningType {
		for _, scanning := range j.spec.Scannings {
			jobTask, err := j.toJobTask(scanning, taskID, string(j.spec.ScanningType), "", "", logger)
			if err != nil {
				return nil, err
			}
			resp = append(resp, jobTask)
		}
	}

	// get deploy info from previous build job
	if j.spec.Source == config.SourceFromJob {
		// adapt to the front end, use the direct quoted job name
		if j.spec.OriginJobName != "" {
			j.spec.JobName = j.spec.OriginJobName
		}
		targets, err := j.getOriginReferedJobTargets(j.spec.JobName)
		if err != nil {
			return resp, fmt.Errorf("get origin refered job: %s targets failed, err: %v", j.spec.JobName, err)
		}
		// clear service and image list to prevent old data from remaining
		j.spec.TargetServices = targets
	}

	if j.spec.ScanningType == config.ServiceScanningType {
		for _, target := range j.spec.TargetServices {
			for _, scanning := range j.spec.ServiceAndScannings {
				if scanning.ServiceName != target.ServiceName || scanning.ServiceModule != target.ServiceModule {
					continue
				}
				jobTask, err := j.toJobTask(&scanning.ScanningModule, taskID, string(j.spec.ScanningType), scanning.ServiceName, scanning.ServiceModule, logger)
				if err != nil {
					return resp, err
				}
				resp = append(resp, jobTask)
			}
		}
	}

	j.job.Spec = j.spec
	return resp, nil
}

func (j *ScanningJob) LintJob() error {
	return nil
}

func (j *ScanningJob) GetOutPuts(log *zap.SugaredLogger) []string {
	resp := []string{}
	j.spec = &commonmodels.ZadigScanningJobSpec{}
	if err := commonmodels.IToiYaml(j.job.Spec, j.spec); err != nil {
		return resp
	}
	scanningNames := []string{}
	for _, scanning := range j.spec.Scannings {
		scanningNames = append(scanningNames, scanning.Name)
	}
	scanningInfos, _, err := commonrepo.NewScanningColl().List(&commonrepo.ScanningListOption{ScanningNames: scanningNames, ProjectName: j.workflow.Project}, 0, 0)
	if err != nil {
		log.Errorf("list scanning info failed: %v", err)
		return resp
	}
	for _, scanningInfo := range scanningInfos {
		jobKey := strings.Join([]string{j.job.Name, scanningInfo.Name}, ".")
		if j.spec.ScanningType == config.ServiceScanningType {
			for _, target := range j.spec.TargetServices {
				for _, scanning := range j.spec.ServiceAndScannings {
					if scanning.ServiceName != target.ServiceName || scanning.ServiceModule != target.ServiceModule {
						continue
					}
					jobKey = strings.Join([]string{j.job.Name, scanning.Name, target.ServiceName, target.ServiceModule}, ".")
				}
			}
		}
		resp = append(resp, getOutputKey(jobKey, ensureScanningOutputs(scanningInfo.Outputs))...)
	}
	return resp
}

func (j *ScanningJob) toJobTask(scanning *commonmodels.ScanningModule, taskID int64, scanningType, serviceName, serviceModule string, logger *zap.SugaredLogger) (*commonmodels.JobTask, error) {
	scanningInfo, err := commonrepo.NewScanningColl().Find(j.workflow.Project, scanning.Name)
	if err != nil {
		return nil, fmt.Errorf("find scanning: %s error: %v", scanning.Name, err)
	}

	if err := fillScanningDetail(scanningInfo); err != nil {
		return nil, err
	}

	basicImage, err := commonrepo.NewBasicImageColl().Find(scanningInfo.ImageID)
	if err != nil {
		return nil, fmt.Errorf("find basic image: %s error: %v", scanningInfo.ImageID, err)
	}
	registries, err := commonservice.ListRegistryNamespaces("", true, logger)
	if err != nil {
		return nil, fmt.Errorf("list registries error: %v", err)
	}
	var timeout int64
	if scanningInfo.AdvancedSetting != nil {
		timeout = scanningInfo.AdvancedSetting.Timeout
	}
	jobTaskSpec := new(commonmodels.JobTaskFreestyleSpec)

	jobInfo := map[string]string{
		JobNameKey:      j.job.Name,
		"scanning_name": scanning.Name,
		"scanning_type": scanningType,
	}

	if scanningType == string(config.ServiceScanningType) {
		jobInfo["service_name"] = serviceName
		jobInfo["service_module"] = serviceModule
	}

	jobKey := strings.Join([]string{j.job.Name, scanning.Name}, ".")
	if scanningType == string(config.ServiceScanningType) {
		jobKey = strings.Join([]string{j.job.Name, scanning.Name, serviceName, serviceModule}, ".")
	}

	jobTask := &commonmodels.JobTask{
		Name:           jobNameFormat(strings.Join([]string{scanning.Name, j.job.Name, serviceName, serviceModule}, "-")),
		Key:            jobKey,
		JobInfo:        jobInfo,
		JobType:        string(config.JobZadigScanning),
		Spec:           jobTaskSpec,
		Timeout:        timeout,
		Outputs:        ensureScanningOutputs(scanningInfo.Outputs),
		Infrastructure: scanningInfo.Infrastructure,
		VMLabels:       scanningInfo.VMLabels,
		ErrorPolicy:    j.job.ErrorPolicy,
	}
	envs := getScanningJobVariables(scanning.Repos, taskID, j.workflow.Project, j.workflow.Name, j.workflow.DisplayName, jobTask.Infrastructure, scanningType, serviceName, serviceModule, scanning.Name)
	envs = append(envs, scanningInfo.Envs...)

	scanningImage := basicImage.Value
	if scanningImage == "sonarsource/sonar-scanner-cli" {
		scanningImage += ":5.0.1"
	}
	if basicImage.ImageFrom == commonmodels.ImageFromKoderover {
		scanningImage = strings.ReplaceAll(config.ReaperImage(), "${BuildOS}", basicImage.Value)
	}
	jobTaskSpec.Properties = commonmodels.JobProperties{
		Timeout:             timeout,
		ResourceRequest:     scanningInfo.AdvancedSetting.ResReq,
		ResReqSpec:          scanningInfo.AdvancedSetting.ResReqSpec,
		ClusterID:           scanningInfo.AdvancedSetting.ClusterID,
		StrategyID:          scanningInfo.AdvancedSetting.StrategyID,
		BuildOS:             scanningImage,
		ImageFrom:           setting.ImageFromCustom,
		Envs:                append(envs, renderKeyVals(scanning.KeyVals, scanningInfo.Envs)...),
		Registries:          registries,
		ShareStorageDetails: getShareStorageDetail(j.workflow.ShareStorages, scanning.ShareStorageInfo, j.workflow.Name, taskID),
	}

	cacheS3 := &commonmodels.S3Storage{}
	clusterInfo, err := commonrepo.NewK8SClusterColl().Get(scanningInfo.AdvancedSetting.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster: %s, error: %v", scanningInfo.AdvancedSetting.ClusterID, err)
	}

	if jobTask.Infrastructure == setting.JobVMInfrastructure {
		jobTaskSpec.Properties.CacheEnable = scanningInfo.AdvancedSetting.Cache.CacheEnable
		jobTaskSpec.Properties.CacheDirType = scanningInfo.AdvancedSetting.Cache.CacheDirType
		jobTaskSpec.Properties.CacheUserDir = scanningInfo.AdvancedSetting.Cache.CacheUserDir
	} else {
		if clusterInfo.Cache.MediumType == "" {
			jobTaskSpec.Properties.CacheEnable = false
		} else {
			jobTaskSpec.Properties.Cache = clusterInfo.Cache
			if scanningInfo.AdvancedSetting.Cache != nil {
				jobTaskSpec.Properties.CacheEnable = scanningInfo.AdvancedSetting.Cache.CacheEnable
				jobTaskSpec.Properties.CacheDirType = scanningInfo.AdvancedSetting.Cache.CacheDirType
				jobTaskSpec.Properties.CacheUserDir = scanningInfo.AdvancedSetting.Cache.CacheUserDir

				if jobTaskSpec.Properties.Cache.MediumType == types.ObjectMedium {
					cacheS3, err = commonrepo.NewS3StorageColl().Find(jobTaskSpec.Properties.Cache.ObjectProperties.ID)
					if err != nil {
						return nil, fmt.Errorf("find cache s3 storage: %s error: %v", jobTaskSpec.Properties.Cache.ObjectProperties.ID, err)
					}
				}
			} else {
				jobTaskSpec.Properties.CacheEnable = false
			}
		}
	}

	if len(scanning.Repos) > 0 {
		jobTaskSpec.Properties.Envs = append(jobTaskSpec.Properties.Envs, &commonmodels.KeyVal{
			Key:   "BRANCH",
			Value: scanning.Repos[0].Branch,
		})
	}

	// init tools install step
	tools := []*step.Tool{}
	for _, tool := range scanningInfo.Installs {
		tools = append(tools, &step.Tool{
			Name:    tool.Name,
			Version: tool.Version,
		})
	}
	toolInstallStep := &commonmodels.StepTask{
		Name:     fmt.Sprintf("%s-%s", scanning.Name, "tool-install"),
		JobName:  jobTask.Name,
		StepType: config.StepTools,
		Spec:     step.StepToolInstallSpec{Installs: tools},
	}
	jobTaskSpec.Steps = append(jobTaskSpec.Steps, toolInstallStep)

	// init download object cache step
	if jobTaskSpec.Properties.CacheEnable && jobTaskSpec.Properties.Cache.MediumType == types.ObjectMedium {
		cacheDir := "/workspace"
		if jobTaskSpec.Properties.CacheDirType == types.UserDefinedCacheDir {
			cacheDir = jobTaskSpec.Properties.CacheUserDir
		}
		downloadArchiveStep := &commonmodels.StepTask{
			Name:     fmt.Sprintf("%s-%s", scanning.Name, "download-archive"),
			JobName:  jobTask.Name,
			StepType: config.StepDownloadArchive,
			Spec: step.StepDownloadArchiveSpec{
				UnTar:      true,
				IgnoreErr:  true,
				FileName:   setting.ScanningOSSCacheFileName,
				ObjectPath: getScanningJobCacheObjectPath(j.workflow.Name, scanning.Name),
				DestDir:    cacheDir,
				S3:         modelS3toS3(cacheS3),
			},
		}
		jobTaskSpec.Steps = append(jobTaskSpec.Steps, downloadArchiveStep)
	}

	// init git clone step
	gitStep := &commonmodels.StepTask{
		Name:     scanning.Name + "-git",
		JobName:  jobTask.Name,
		StepType: config.StepGit,
		Spec:     step.StepGitSpec{Repos: renderRepos(scanning.Repos, scanningInfo.Repos, jobTaskSpec.Properties.Envs)},
	}
	jobTaskSpec.Steps = append(jobTaskSpec.Steps, gitStep)
	repoName := ""
	branch := ""
	if len(scanningInfo.Repos) > 0 {
		if scanningInfo.Repos[0].CheckoutPath != "" {
			repoName = scanningInfo.Repos[0].CheckoutPath
		} else {
			repoName = scanningInfo.Repos[0].RepoName
		}

		branch = scanningInfo.Repos[0].Branch
	}
	// init debug before step
	debugBeforeStep := &commonmodels.StepTask{
		Name:     scanning.Name + "-debug-before",
		JobName:  jobTask.Name,
		StepType: config.StepDebugBefore,
	}
	jobTaskSpec.Steps = append(jobTaskSpec.Steps, debugBeforeStep)
	// init script step
	if scanningInfo.ScannerType == types.ScanningTypeSonar {
		scriptStep := &commonmodels.StepTask{
			JobName: jobTask.Name,
		}
		if scanningInfo.ScriptType == types.ScriptTypeShell || scanningInfo.ScriptType == "" {
			scriptStep.Name = scanning.Name + "-shell"
			scriptStep.StepType = config.StepShell
			scriptStep.Spec = &step.StepShellSpec{
				Scripts:     append(strings.Split(replaceWrapLine(scanningInfo.Script), "\n"), outputScript(scanningInfo.Outputs, jobTask.Infrastructure)...),
				SkipPrepare: true,
			}
		} else if scanningInfo.ScriptType == types.ScriptTypeBatchFile {
			scriptStep.Name = scanning.Name + "-batchfile"
			scriptStep.StepType = config.StepBatchFile
			scriptStep.Spec = &step.StepBatchFileSpec{
				Scripts:     append(strings.Split(replaceWrapLine(scanningInfo.Script), "\n"), outputScript(scanningInfo.Outputs, jobTask.Infrastructure)...),
				SkipPrepare: true,
			}
		} else if scanningInfo.ScriptType == types.ScriptTypePowerShell {
			scriptStep.Name = scanning.Name + "-powershell"
			scriptStep.StepType = config.StepPowerShell
			scriptStep.Spec = &step.StepPowerShellSpec{
				Scripts:     append(strings.Split(replaceWrapLine(scanningInfo.Script), "\n"), outputScript(scanningInfo.Outputs, jobTask.Infrastructure)...),
				SkipPrepare: true,
			}
		}
		jobTaskSpec.Steps = append(jobTaskSpec.Steps, scriptStep)

		sonarInfo, err := commonrepo.NewSonarIntegrationColl().GetByID(context.TODO(), scanningInfo.SonarID)
		if err != nil {
			return nil, fmt.Errorf("failed to get sonar integration information to create scanning task, error: %s", err)

		}

		projectKey := renderEnv(sonar.GetSonarProjectKeyFromConfig(scanningInfo.Parameter), jobTaskSpec.Properties.Envs)
		resultAddr, err := sonar.GetSonarAddressWithProjectKey(sonarInfo.ServerAddress, projectKey)
		if err != nil {
			log.Errorf("failed to get sonar address with project key, error: %s", err)
		}

		sonarLinkKeyVal := &commonmodels.KeyVal{
			Key:   "SONAR_LINK",
			Value: resultAddr,
		}
		jobTaskSpec.Properties.Envs = append(jobTaskSpec.Properties.Envs, sonarLinkKeyVal)
		jobTaskSpec.Properties.Envs = append(jobTaskSpec.Properties.Envs, &commonmodels.KeyVal{
			Key:          "SONAR_TOKEN",
			Value:        sonarInfo.Token,
			IsCredential: true,
		})

		jobTaskSpec.Properties.Envs = append(jobTaskSpec.Properties.Envs, &commonmodels.KeyVal{
			Key:   "SONAR_URL",
			Value: sonarInfo.ServerAddress,
		})

		if scanningInfo.EnableScanner {
			sonarScriptStep := &commonmodels.StepTask{
				JobName: jobTask.Name,
			}
			if scanningInfo.ScriptType == types.ScriptTypeShell || scanningInfo.ScriptType == "" {
				sonarConfig := fmt.Sprintf("sonar.login=%s\nsonar.host.url=%s\n%s", sonarInfo.Token, sonarInfo.ServerAddress, scanningInfo.Parameter)
				sonarConfig = strings.ReplaceAll(sonarConfig, "$branch", branch)
				sonarScript := fmt.Sprintf("set -e\ncd %s\ncat > sonar-project.properties << EOF\n%s\nEOF\nsonar-scanner", repoName, renderEnv(sonarConfig, jobTaskSpec.Properties.Envs))

				sonarScriptStep.Name = scanning.Name + "-sonar-shell"
				sonarScriptStep.StepType = config.StepShell
				sonarScriptStep.Spec = &step.StepShellSpec{
					Scripts:     strings.Split(replaceWrapLine(sonarScript), "\n"),
					SkipPrepare: true,
				}
			} else if scanningInfo.ScriptType == types.ScriptTypeBatchFile {
				sonarScript := fmt.Sprintf("@echo off\nsetlocal enabledelayedexpansion\ncd %s\n\n", repoName)
				sonarScript += "(\n"

				sonarConfig := fmt.Sprintf("sonar.login=%s\nsonar.host.url=%s\n%s", sonarInfo.Token, sonarInfo.ServerAddress, scanningInfo.Parameter)
				sonarConfig = strings.ReplaceAll(sonarConfig, "$branch", branch)
				sonarConfig = renderEnv(sonarConfig, jobTaskSpec.Properties.Envs)
				sonarConfigArr := strings.Split(sonarConfig, "\n")
				for _, config := range sonarConfigArr {
					sonarScript += fmt.Sprintf("echo %s\n", config)
				}

				sonarScript += "\n) > sonar-project.properties\n\nsonar-scanner\n\nendlocal"
				sonarScriptStep.Name = scanning.Name + "-sonar-batchfile"
				sonarScriptStep.StepType = config.StepBatchFile
				sonarScriptStep.Spec = &step.StepBatchFileSpec{
					Scripts:     strings.Split(replaceWrapLine(sonarScript), "\n"),
					SkipPrepare: true,
				}
			} else if scanningInfo.ScriptType == types.ScriptTypePowerShell {
				sonarScript := fmt.Sprintf("Set-StrictMode -Version Latest\nSet-Location -Path \"%s\"\n", repoName)
				sonarScript += "@\"\n"

				sonarConfig := fmt.Sprintf("sonar.login=%s\nsonar.host.url=%s\n%s", sonarInfo.Token, sonarInfo.ServerAddress, scanningInfo.Parameter)
				sonarConfig = strings.ReplaceAll(sonarConfig, "$branch", branch)
				sonarConfig = renderEnv(sonarConfig, jobTaskSpec.Properties.Envs)
				sonarConfigArr := strings.Split(sonarConfig, "\n")
				for _, config := range sonarConfigArr {
					sonarScript += fmt.Sprintf("%s\n", config)
				}

				sonarScript += "\"@ | Out-File -FilePath \"sonar-project.properties\" -Encoding UTF8\n\nsonar-scanner"
				sonarScriptStep.Name = scanning.Name + "-sonar-powershell"
				sonarScriptStep.StepType = config.StepPowerShell
				sonarScriptStep.Spec = &step.StepPowerShellSpec{
					Scripts:     strings.Split(replaceWrapLine(sonarScript), "\n"),
					SkipPrepare: true,
				}
			}

			jobTaskSpec.Steps = append(jobTaskSpec.Steps, sonarScriptStep)

		}

		sonarGetMetricsStep := &commonmodels.StepTask{
			Name:     scanning.Name + "-sonar-get-metrics",
			JobName:  jobTask.Name,
			JobKey:   jobTask.Key,
			StepType: config.StepSonarGetMetrics,
			Spec: &step.StepSonarGetMetricsSpec{
				ProjectKey:       projectKey,
				Parameter:        scanningInfo.Parameter,
				CheckDir:         repoName,
				SonarToken:       sonarInfo.Token,
				SonarServer:      sonarInfo.ServerAddress,
				CheckQualityGate: scanningInfo.CheckQualityGate,
			},
		}
		jobTaskSpec.Steps = append(jobTaskSpec.Steps, sonarGetMetricsStep)

		if scanningInfo.CheckQualityGate {
			sonarChekStep := &commonmodels.StepTask{
				Name:     scanning.Name + "-sonar-check",
				JobName:  jobTask.Name,
				JobKey:   jobTask.Key,
				StepType: config.StepSonarCheck,
				Spec: &step.StepSonarCheckSpec{
					ProjectKey:  projectKey,
					Parameter:   scanningInfo.Parameter,
					CheckDir:    repoName,
					SonarToken:  sonarInfo.Token,
					SonarServer: sonarInfo.ServerAddress,
				},
			}
			jobTaskSpec.Steps = append(jobTaskSpec.Steps, sonarChekStep)
		}
	} else {
		scriptStep := &commonmodels.StepTask{
			JobName: jobTask.Name,
		}
		if scanningInfo.ScriptType == types.ScriptTypeShell || scanningInfo.ScriptType == "" {
			scriptStep.Name = scanning.Name + "-shell"
			scriptStep.StepType = config.StepShell
			scriptStep.Spec = &step.StepShellSpec{
				Scripts: append(strings.Split(replaceWrapLine(scanningInfo.Script), "\n"), outputScript(scanningInfo.Outputs, jobTask.Infrastructure)...),
			}
		} else if scanningInfo.ScriptType == types.ScriptTypeBatchFile {
			scriptStep.Name = scanning.Name + "-batchfile"
			scriptStep.StepType = config.StepBatchFile
			scriptStep.Spec = &step.StepBatchFileSpec{
				Scripts: append(strings.Split(replaceWrapLine(scanningInfo.Script), "\n"), outputScript(scanningInfo.Outputs, jobTask.Infrastructure)...),
			}
		} else if scanningInfo.ScriptType == types.ScriptTypePowerShell {
			scriptStep.Name = scanning.Name + "-powershell"
			scriptStep.StepType = config.StepPowerShell
			scriptStep.Spec = &step.StepPowerShellSpec{
				Scripts: append(strings.Split(replaceWrapLine(scanningInfo.Script), "\n"), outputScript(scanningInfo.Outputs, jobTask.Infrastructure)...),
			}
		}
		jobTaskSpec.Steps = append(jobTaskSpec.Steps, scriptStep)
	}
	// init debug after step
	debugAfterStep := &commonmodels.StepTask{
		Name:     scanning.Name + "-debug-after",
		JobName:  jobTask.Name,
		StepType: config.StepDebugAfter,
	}

	// init object cache step
	if jobTaskSpec.Properties.CacheEnable && jobTaskSpec.Properties.Cache.MediumType == types.ObjectMedium {
		cacheDir := "/workspace"
		if jobTaskSpec.Properties.CacheDirType == types.UserDefinedCacheDir {
			cacheDir = jobTaskSpec.Properties.CacheUserDir
		}
		tarArchiveStep := &commonmodels.StepTask{
			Name:     fmt.Sprintf("%s-%s", scanning.Name, "tar-archive"),
			JobName:  jobTask.Name,
			StepType: config.StepTarArchive,
			Spec: step.StepTarArchiveSpec{
				FileName:     setting.ScanningOSSCacheFileName,
				ResultDirs:   []string{"."},
				AbsResultDir: true,
				TarDir:       cacheDir,
				ChangeTarDir: true,
				S3DestDir:    getScanningJobCacheObjectPath(j.workflow.Name, scanning.Name),
				IgnoreErr:    true,
				S3Storage:    modelS3toS3(cacheS3),
			},
		}
		jobTaskSpec.Steps = append(jobTaskSpec.Steps, tarArchiveStep)
	}

	jobTaskSpec.Steps = append(jobTaskSpec.Steps, debugAfterStep)

	return jobTask, nil
}

func (j *ScanningJob) getOriginReferedJobTargets(jobName string) ([]*commonmodels.ServiceTestTarget, error) {
	servicetargets := []*commonmodels.ServiceTestTarget{}
	for _, stage := range j.workflow.Stages {
		for _, job := range stage.Jobs {
			if job.Name != j.spec.JobName {
				continue
			}
			if job.JobType == config.JobZadigBuild {
				buildSpec := &commonmodels.ZadigBuildJobSpec{}
				if err := commonmodels.IToi(job.Spec, buildSpec); err != nil {
					return servicetargets, err
				}
				for _, build := range buildSpec.ServiceAndBuilds {
					servicetargets = append(servicetargets, &commonmodels.ServiceTestTarget{
						ServiceName:   build.ServiceName,
						ServiceModule: build.ServiceModule,
					})
				}
				return servicetargets, nil
			}
			if job.JobType == config.JobZadigDistributeImage {
				distributeSpec := &commonmodels.ZadigDistributeImageJobSpec{}
				if err := commonmodels.IToi(job.Spec, distributeSpec); err != nil {
					return servicetargets, err
				}
				for _, distribute := range distributeSpec.Targets {
					servicetargets = append(servicetargets, &commonmodels.ServiceTestTarget{
						ServiceName:   distribute.ServiceName,
						ServiceModule: distribute.ServiceModule,
					})
				}
				return servicetargets, nil
			}
			if job.JobType == config.JobZadigDeploy {
				deploySpec := &commonmodels.ZadigDeployJobSpec{}
				if err := commonmodels.IToi(job.Spec, deploySpec); err != nil {
					return servicetargets, err
				}
				for _, svc := range deploySpec.Services {
					for _, module := range svc.Modules {
						servicetargets = append(servicetargets, &commonmodels.ServiceTestTarget{
							ServiceName:   svc.ServiceName,
							ServiceModule: module.ServiceModule,
						})
					}
				}
				return servicetargets, nil
			}
			if job.JobType == config.JobZadigScanning {
				scanningSpec := &commonmodels.ZadigScanningJobSpec{}
				if err := commonmodels.IToi(job.Spec, scanningSpec); err != nil {
					return servicetargets, err
				}
				servicetargets = scanningSpec.TargetServices
				return servicetargets, nil
			}
		}
	}
	return nil, fmt.Errorf("build job %s not found", jobName)
}

func getScanningJobVariables(repos []*types.Repository, taskID int64, project, workflowName, workflowDisplayName, infrastructure, scanningType, serviceName, serviceModule, scanningName string) []*commonmodels.KeyVal {
	ret := []*commonmodels.KeyVal{}

	// basic envs
	ret = append(ret, PrepareDefaultWorkflowTaskEnvs(project, workflowName, workflowDisplayName, infrastructure, taskID)...)
	// repo envs
	ret = append(ret, getReposVariables(repos)...)

	ret = append(ret, &commonmodels.KeyVal{Key: "SCANNING_TYPE", Value: scanningType, IsCredential: false})
	ret = append(ret, &commonmodels.KeyVal{Key: "SERVICE_NAME", Value: serviceName, IsCredential: false})
	ret = append(ret, &commonmodels.KeyVal{Key: "SERVICE_MODULE", Value: serviceModule, IsCredential: false})
	ret = append(ret, &commonmodels.KeyVal{Key: "SCANNING_NAME", Value: scanningName, IsCredential: false})

	return ret
}

func fillScanningDetail(moduleScanning *commonmodels.Scanning) error {
	if moduleScanning.TemplateID == "" {
		return nil
	}
	templateInfo, err := commonrepo.NewScanningTemplateColl().Find(&commonrepo.ScanningTemplateQueryOption{
		ID: moduleScanning.TemplateID,
	})
	if err != nil {
		return fmt.Errorf("failed to find scanning template with id: %s, err: %s", moduleScanning.TemplateID, err)
	}

	moduleScanning.Infrastructure = templateInfo.Infrastructure
	moduleScanning.VMLabels = templateInfo.VMLabels
	moduleScanning.ScannerType = templateInfo.ScannerType
	moduleScanning.EnableScanner = templateInfo.EnableScanner
	moduleScanning.ImageID = templateInfo.ImageID
	moduleScanning.SonarID = templateInfo.SonarID
	moduleScanning.Installs = templateInfo.Installs
	moduleScanning.Parameter = templateInfo.Parameter
	moduleScanning.Envs = commonservice.MergeBuildEnvs(templateInfo.Envs, moduleScanning.Envs)
	moduleScanning.ScriptType = templateInfo.ScriptType
	moduleScanning.Script = templateInfo.Script
	moduleScanning.AdvancedSetting = templateInfo.AdvancedSetting
	moduleScanning.CheckQualityGate = templateInfo.CheckQualityGate

	return nil
}

func getScanningJobCacheObjectPath(workflowName, scanningName string) string {
	return fmt.Sprintf("%s/cache/%s", workflowName, scanningName)
}

func ensureScanningOutputs(outputs []*commonmodels.Output) []*commonmodels.Output {
	keyMap := map[string]struct{}{}
	for _, output := range outputs {
		keyMap[output.Name] = struct{}{}
	}
	if _, ok := keyMap[setting.WorkflowScanningJobOutputKey]; !ok {
		outputs = append(outputs, &commonmodels.Output{
			Name: setting.WorkflowScanningJobOutputKey,
		})
	}
	if _, ok := keyMap[setting.WorkflowScanningJobOutputKeyProject]; !ok {
		outputs = append(outputs, &commonmodels.Output{
			Name: setting.WorkflowScanningJobOutputKeyProject,
		})
	}
	return outputs
}
