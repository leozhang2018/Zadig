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
	"fmt"
	"strings"

	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	"github.com/koderover/zadig/v2/pkg/util"
	"go.uber.org/zap"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonservice "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service/repository"
	"github.com/koderover/zadig/v2/pkg/setting"
	"github.com/koderover/zadig/v2/pkg/tool/log"
	"github.com/koderover/zadig/v2/pkg/types/job"
	"github.com/koderover/zadig/v2/pkg/types/step"
)

const (
	DistributeTimeout int64 = 10

	// WorkflowInputImageTagVariable PreBuildImageTagVariable
	// These variables are not really workflow variables, will convert to real value or workflow variables
	// Not return from GetWorkflowGlobalVars function, instead of frontend
	WorkflowInputImageTagVariable = "{{.workflow.input.imageTag}}"
	PreBuildImageTagVariable      = "{{.job.preBuild.imageTag}}"
	PreJobImageTagVariable        = "{{.job.preJob.imageTag}}"
)

type ImageDistributeJob struct {
	job      *commonmodels.Job
	workflow *commonmodels.WorkflowV4
	spec     *commonmodels.ZadigDistributeImageJobSpec
}

func (j *ImageDistributeJob) Instantiate() error {
	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToiYaml(j.job.Spec, j.spec); err != nil {
		return err
	}
	j.job.Spec = j.spec
	return nil
}

func (j *ImageDistributeJob) SetPreset() error {
	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}

	if j.spec.Source == config.SourceFromJob {
		serviceReferredJob := getOriginJobName(j.workflow, j.spec.JobName)

		targets, _, err := j.getOriginReferredJobTargets(serviceReferredJob, j.spec.JobName)
		if err != nil {
			return fmt.Errorf("failed to get referred job info for distribute job: %s, error: %s", j.job.Name, err)
		}

		j.spec.Targets = targets
	} else if j.spec.Source == config.SourceRuntime {
		servicesMap, err := repository.GetMaxRevisionsServicesMap(j.workflow.Project, false)
		if err != nil {
			return fmt.Errorf("get services map error: %v", err)
		}

		for _, target := range j.spec.Targets {
			target.ImageName = target.ServiceModule

			service, ok := servicesMap[target.ServiceName]
			if !ok {
				log.Errorf("service %s not found", target.ServiceName)
				continue
			}

			for _, container := range service.Containers {
				if container.Name == target.ServiceModule {
					target.ImageName = container.ImageName
					break
				}
			}
		}
	}

	j.job.Spec = j.spec
	return nil
}

func (j *ImageDistributeJob) SetOptions() error {
	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}

	servicesMap, err := repository.GetMaxRevisionsServicesMap(j.workflow.Project, false)
	if err != nil {
		return fmt.Errorf("get services map error: %v", err)
	}

	options := make([]*commonmodels.DistributeTarget, 0)
	for _, svc := range servicesMap {
		for _, module := range svc.Containers {
			options = append(options, &commonmodels.DistributeTarget{
				ServiceName:   svc.ServiceName,
				ServiceModule: module.Name,
				ImageName:     util.ExtractImageName(module.Image),
			})
		}
	}

	j.spec.TargetOptions = options
	j.job.Spec = j.spec
	return nil
}

func (j *ImageDistributeJob) ClearSelectionField() error {
	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}

	j.spec.Targets = make([]*commonmodels.DistributeTarget, 0)
	j.job.Spec = j.spec
	return nil
}

func (j *ImageDistributeJob) MergeArgs(args *commonmodels.Job) error {
	if j.job.Name == args.Name && j.job.JobType == args.JobType {
		j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
		if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
			return err
		}
		argsSpec := &commonmodels.ZadigDistributeImageJobSpec{}
		if err := commonmodels.IToi(args.Spec, argsSpec); err != nil {
			return err
		}
		j.spec.Targets = argsSpec.Targets
		j.job.Spec = j.spec
	}
	return nil
}

func (j *ImageDistributeJob) UpdateWithLatestSetting() error {
	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}

	latestWorkflow, err := commonrepo.NewWorkflowV4Coll().Find(j.workflow.Name)
	if err != nil {
		log.Errorf("Failed to find original workflow to set options, error: %s", err)
	}

	latestSpec := new(commonmodels.ZadigDistributeImageJobSpec)
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

	// source is a bit tricky: if the saved args has a source of fromjob, but it has been change to runtime in the config
	// we need to not only update its source but also set services to empty slice.
	if j.spec.Source == config.SourceFromJob && latestSpec.Source == config.SourceRuntime {
		j.spec.Targets = make([]*commonmodels.DistributeTarget, 0)
	}
	j.spec.Source = latestSpec.Source

	if j.spec.Source == config.SourceFromJob {
		j.spec.JobName = latestSpec.JobName
	} else {
		j.spec.SourceRegistryID = latestSpec.SourceRegistryID
	}

	j.spec.TargetRegistryID = latestSpec.TargetRegistryID
	j.spec.Timeout = latestSpec.Timeout
	j.spec.ClusterID = latestSpec.ClusterID
	j.spec.StrategyID = latestSpec.StrategyID
	j.spec.EnableTargetImageTagRule = latestSpec.EnableTargetImageTagRule
	j.spec.TargetImageTagRule = latestSpec.TargetImageTagRule
	j.job.Spec = j.spec
	return nil
}

func (j *ImageDistributeJob) ToJobs(taskID int64) ([]*commonmodels.JobTask, error) {
	logger := log.SugaredLogger()
	resp := []*commonmodels.JobTask{}

	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return resp, err
	}

	sourceReg, err := commonservice.FindRegistryById(j.spec.SourceRegistryID, true, logger)
	if err != nil {
		return resp, fmt.Errorf("source image registry: %s not found: %v", j.spec.SourceRegistryID, err)
	}
	targetReg, err := commonservice.FindRegistryById(j.spec.TargetRegistryID, true, logger)
	if err != nil {
		return resp, fmt.Errorf("target image registry: %s not found: %v", j.spec.TargetRegistryID, err)
	}

	switch j.spec.Source {
	case config.SourceFromJob:
		serviceReferredJob := getOriginJobName(j.workflow, j.spec.JobName)

		targets, registryID, err := j.getOriginReferredJobTargets(serviceReferredJob, j.spec.JobName)
		if err != nil {
			return nil, fmt.Errorf("failed to get referred job info for distribute job: %s, error: %s", j.job.Name, err)
		}

		j.spec.SourceRegistryID = registryID

		targetTagMap := map[string]commonmodels.DistributeTarget{}
		for _, target := range j.spec.Targets {
			targetTagMap[getServiceKey(target.ServiceName, target.ServiceModule)] = *target
		}

		for _, target := range targets {
			if j.spec.EnableTargetImageTagRule {
				target.TargetTag = strings.ReplaceAll(j.spec.TargetImageTagRule, PreBuildImageTagVariable,
					fmt.Sprintf("{{.job.%s.%s.%s.output.%s}}", j.spec.JobName, target.ServiceName, target.ServiceModule, IMAGETAGKEY))
				target.TargetTag = strings.ReplaceAll(j.spec.TargetImageTagRule, PreJobImageTagVariable,
					fmt.Sprintf("{{.job.%s.%s.%s.output.%s}}", j.spec.JobName, target.ServiceName, target.ServiceModule, IMAGETAGKEY))
				target.UpdateTag = true
			} else {
				target.TargetTag = targetTagMap[getServiceKey(target.ServiceName, target.ServiceModule)].TargetTag
				target.UpdateTag = targetTagMap[getServiceKey(target.ServiceName, target.ServiceModule)].UpdateTag
			}
		}

		j.spec.Targets = targets
	case config.SourceRuntime:
		for _, target := range j.spec.Targets {
			if target.ImageName == "" {
				target.SourceImage = getImage(target.ServiceModule, target.SourceTag, sourceReg)
			} else {
				target.SourceImage = getImage(target.ImageName, target.SourceTag, sourceReg)
			}
			if j.spec.EnableTargetImageTagRule {
				target.TargetTag = strings.ReplaceAll(j.spec.TargetImageTagRule,
					WorkflowInputImageTagVariable, target.SourceTag)
			}
			target.UpdateTag = true
		}
	}

	stepSpec := &step.StepImageDistributeSpec{
		SourceRegistry: getRegistry(sourceReg),
		TargetRegistry: getRegistry(targetReg),
	}
	for _, target := range j.spec.Targets {
		// for other job refer current latest image.
		targetKey := strings.Join([]string{j.job.Name, target.ServiceName, target.ServiceModule}, ".")
		target.TargetImage = job.GetJobOutputKey(targetKey, "IMAGE")

		stepSpec.DistributeTarget = append(stepSpec.DistributeTarget, &step.DistributeTaskTarget{
			SourceImage:   target.SourceImage,
			ServiceName:   target.ServiceName,
			ServiceModule: target.ServiceModule,
			TargetTag:     target.TargetTag,
			UpdateTag:     target.UpdateTag,
		})
	}

	jobTaskSpec := &commonmodels.JobTaskFreestyleSpec{
		Properties: commonmodels.JobProperties{
			Timeout:         j.spec.Timeout,
			ResourceRequest: setting.MinRequest,
			ClusterID:       j.spec.ClusterID,
			StrategyID:      j.spec.StrategyID,
			BuildOS:         "focal",
			ImageFrom:       commonmodels.ImageFromKoderover,
		},
		Steps: []*commonmodels.StepTask{
			{
				Name:     "distribute",
				StepType: config.StepDistributeImage,
				Spec:     stepSpec,
			},
		},
	}
	jobTask := &commonmodels.JobTask{
		Name: j.job.Name,
		Key:  j.job.Name,
		JobInfo: map[string]string{
			JobNameKey: j.job.Name,
		},
		JobType:     string(config.JobZadigDistributeImage),
		Spec:        jobTaskSpec,
		Timeout:     getTimeout(j.spec.Timeout),
		ErrorPolicy: j.job.ErrorPolicy,
	}
	resp = append(resp, jobTask)
	j.job.Spec = j.spec
	return resp, nil
}

func (j *ImageDistributeJob) LintJob() error {
	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToiYaml(j.job.Spec, j.spec); err != nil {
		return err
	}
	if j.spec.Source != config.SourceFromJob {
		return nil
	}
	jobRankMap := getJobRankMap(j.workflow.Stages)
	buildJobRank, ok := jobRankMap[j.spec.JobName]
	if !ok || buildJobRank >= jobRankMap[j.job.Name] {
		return fmt.Errorf("can not quote job %s in job %s", j.spec.JobName, j.job.Name)
	}
	return nil
}

func getQuoteBuildJobSpec(jobName string, workflow *commonmodels.WorkflowV4) (*commonmodels.ZadigBuildJobSpec, error) {
	resp := &commonmodels.ZadigBuildJobSpec{}
	for _, stage := range workflow.Stages {
		for _, job := range stage.Jobs {
			if job.Name != jobName {
				continue
			}
			if job.JobType != config.JobZadigBuild {
				return resp, fmt.Errorf("cannot reference job: %s that is not a build", jobName)
			}
			if err := commonmodels.IToi(job.Spec, resp); err != nil {
				return resp, err
			}
			return resp, nil
		}
	}
	return resp, fmt.Errorf("reference job: %s not found", jobName)
}

func (j *ImageDistributeJob) getOriginReferredJobTargets(serviceReferredJob, imageReferredJob string) ([]*commonmodels.DistributeTarget, string, error) {
	servicetargets := []*commonmodels.DistributeTarget{}
	var sourceRegistryID string
	found := false
serviceLoop:
	for _, stage := range j.workflow.Stages {
		for _, job := range stage.Jobs {
			if job.Name != serviceReferredJob {
				continue
			}
			if job.JobType == config.JobZadigBuild {
				buildSpec := &commonmodels.ZadigBuildJobSpec{}
				if err := commonmodels.IToi(job.Spec, buildSpec); err != nil {
					return nil, "", fmt.Errorf("failed to decode build job spec, error: %s", err)
				}
				for _, build := range buildSpec.ServiceAndBuilds {
					servicetargets = append(servicetargets, &commonmodels.DistributeTarget{
						ServiceName:   build.ServiceName,
						ServiceModule: build.ServiceModule,
					})
				}
				sourceRegistryID = buildSpec.DockerRegistryID
				found = true
				break serviceLoop
			}

			if job.JobType == config.JobZadigDistributeImage {
				distributeSpec := &commonmodels.ZadigDistributeImageJobSpec{}
				if err := commonmodels.IToi(job.Spec, distributeSpec); err != nil {
					return nil, "", fmt.Errorf("failed to decode distribute job spec, error: %s", err)
				}
				for _, distribute := range distributeSpec.Targets {
					servicetargets = append(servicetargets, &commonmodels.DistributeTarget{
						ServiceName:   distribute.ServiceName,
						ServiceModule: distribute.ServiceModule,
					})
				}
				sourceRegistryID = distributeSpec.TargetRegistryID
				found = true
				break serviceLoop
			}

			if job.JobType == config.JobZadigDeploy {
				deploySpec := &commonmodels.ZadigDeployJobSpec{}
				if err := commonmodels.IToi(job.Spec, deploySpec); err != nil {
					return nil, "", fmt.Errorf("failed to decode deploy job spec, error: %s", err)
				}
				for _, svc := range deploySpec.Services {
					for _, module := range svc.Modules {
						servicetargets = append(servicetargets, &commonmodels.DistributeTarget{
							ServiceName:   svc.ServiceName,
							ServiceModule: module.ServiceModule,
						})
					}
				}

				envInfo, err := commonrepo.NewProductColl().Find(&commonrepo.ProductFindOptions{
					EnvName: deploySpec.Env,
					Name:    j.workflow.Project,
				})

				if err != nil {
					return nil, "", fmt.Errorf("failed to get deploy job %s env's registry info, error: %s", deploySpec.Env, err)
				}

				sourceRegistryID = envInfo.RegistryID
				found = true
				break serviceLoop
			}
		}
	}

	if !found {
		return nil, "", fmt.Errorf("referred service job %s not found", serviceReferredJob)
	}

	// then we determine the image for the selected job, use the output for each module is enough
	for _, svc := range servicetargets {
		// generate real job keys
		key := job.GetJobOutputKey(fmt.Sprintf("%s.%s.%s", imageReferredJob, svc.ServiceName, svc.ServiceModule), IMAGEKEY)

		svc.SourceImage = key
	}

	return servicetargets, sourceRegistryID, nil
}

func getServiceKey(serviceName, serviceModule string) string {
	return fmt.Sprintf("%s/%s", serviceName, serviceModule)
}

func getImage(name, tag string, reg *commonmodels.RegistryNamespace) string {
	image := fmt.Sprintf("%s/%s:%s", reg.RegAddr, name, tag)
	if len(reg.Namespace) > 0 {
		image = fmt.Sprintf("%s/%s/%s:%s", reg.RegAddr, reg.Namespace, name, tag)
	}
	image = strings.TrimPrefix(image, "http://")
	image = strings.TrimPrefix(image, "https://")
	return image
}

func getRegistry(regDetail *commonmodels.RegistryNamespace) *step.RegistryNamespace {
	reg := &step.RegistryNamespace{
		RegAddr:   regDetail.RegAddr,
		Namespace: regDetail.Namespace,
		AccessKey: regDetail.AccessKey,
		SecretKey: regDetail.SecretKey,
	}
	if regDetail.AdvancedSetting != nil {
		reg.TLSEnabled = regDetail.AdvancedSetting.TLSEnabled
		reg.TLSCert = regDetail.AdvancedSetting.TLSCert
	}
	return reg
}

func getTimeout(timeout int64) int64 {
	if timeout == 0 {
		return DistributeTimeout
	}
	return timeout
}

func (j *ImageDistributeJob) GetOutPuts(log *zap.SugaredLogger) []string {
	resp := []string{}
	j.spec = &commonmodels.ZadigDistributeImageJobSpec{}
	if err := commonmodels.IToiYaml(j.job.Spec, j.spec); err != nil {
		return resp
	}
	for _, target := range j.spec.Targets {
		targetKey := strings.Join([]string{j.job.Name, target.ServiceName, target.ServiceModule}, ".")
		resp = append(resp, getOutputKey(targetKey, []*commonmodels.Output{{Name: "IMAGE"}})...)
	}
	return resp
}
