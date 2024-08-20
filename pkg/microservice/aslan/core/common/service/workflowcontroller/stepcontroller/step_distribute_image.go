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

package stepcontroller

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	"github.com/koderover/zadig/v2/pkg/types/job"
	"github.com/koderover/zadig/v2/pkg/types/step"
)

type distributeImageCtl struct {
	step                *commonmodels.StepTask
	workflowCtx         *commonmodels.WorkflowTaskCtx
	jobName             string
	distributeImageSpec *step.StepImageDistributeSpec
	log                 *zap.SugaredLogger
}

func NewDistributeCtl(stepTask *commonmodels.StepTask, workflowCtx *commonmodels.WorkflowTaskCtx, jobName string, log *zap.SugaredLogger) (*distributeImageCtl, error) {
	yamlString, err := yaml.Marshal(stepTask.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshal image distribute spec error: %v", err)
	}
	distributeSpec := &step.StepImageDistributeSpec{}
	if err := yaml.Unmarshal(yamlString, &distributeSpec); err != nil {
		return nil, fmt.Errorf("unmarshal image distribute error: %v", err)
	}
	stepTask.Spec = distributeSpec
	return &distributeImageCtl{distributeImageSpec: distributeSpec, workflowCtx: workflowCtx, jobName: jobName, log: log, step: stepTask}, nil
}

func (s *distributeImageCtl) PreRun(ctx context.Context) error {
	for _, target := range s.distributeImageSpec.DistributeTarget {
		if target.SourceImage == "" {
			return fmt.Errorf("source image is empty")
		}

		target.SetTargetImage(s.distributeImageSpec.TargetRegistry)
	}
	s.step.Spec = s.distributeImageSpec
	return nil
}

func (s *distributeImageCtl) AfterRun(ctx context.Context) error {
	for _, target := range s.distributeImageSpec.DistributeTarget {
		targetKey := strings.Join([]string{s.jobName, target.ServiceName, target.ServiceModule}, ".")
		s.workflowCtx.GlobalContextSet(job.GetJobOutputKey(targetKey, "IMAGE"), target.TargetImage)
	}
	return nil
}
