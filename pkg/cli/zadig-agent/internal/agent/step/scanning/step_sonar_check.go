/*
Copyright 2024 The KodeRover Authors.

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

package scanning

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/koderover/zadig/v2/pkg/cli/zadig-agent/helper/log"
	"github.com/koderover/zadig/v2/pkg/cli/zadig-agent/internal/common/types"
	"github.com/koderover/zadig/v2/pkg/setting"
	"github.com/koderover/zadig/v2/pkg/tool/sonar"
	"github.com/koderover/zadig/v2/pkg/types/step"
)

type SonarCheckStep struct {
	spec       *step.StepSonarCheckSpec
	envs       []string
	secretEnvs []string
	workspace  string
	dirs       *types.AgentWorkDirs
	Logger     *log.JobLogger
}

func NewSonarCheckStep(spec interface{}, dirs *types.AgentWorkDirs, envs, secretEnvs []string, logger *log.JobLogger) (*SonarCheckStep, error) {
	sonarCheckStep := &SonarCheckStep{dirs: dirs, workspace: dirs.Workspace, envs: envs, secretEnvs: secretEnvs}
	yamlBytes, err := yaml.Marshal(spec)
	if err != nil {
		return sonarCheckStep, fmt.Errorf("marshal spec %+v failed", spec)
	}
	if err := yaml.Unmarshal(yamlBytes, &sonarCheckStep.spec); err != nil {
		return sonarCheckStep, fmt.Errorf("unmarshal spec %s to shell spec failed", yamlBytes)
	}
	sonarCheckStep.Logger = logger
	return sonarCheckStep, nil
}

func (s *SonarCheckStep) Run(ctx context.Context) error {
	s.Logger.Infof("Start check Sonar scanning quality gate status.")
	client := sonar.NewSonarClient(s.spec.SonarServer, s.spec.SonarToken)
	sonarWorkDir := sonar.GetSonarWorkDir(s.spec.Parameter)
	if sonarWorkDir == "" {
		sonarWorkDir = ".scannerwork"
	}
	if !filepath.IsAbs(sonarWorkDir) {
		sonarWorkDir = filepath.Join(s.workspace, s.spec.CheckDir, sonarWorkDir)
	}
	taskReportDir := filepath.Join(sonarWorkDir, "report-task.txt")
	bytes, err := ioutil.ReadFile(taskReportDir)
	if err != nil {
		s.Logger.Errorf("read sonar task report file: %s error :%v", time.Now().Format(setting.WorkflowTimeFormat), taskReportDir, err)
		return err
	}
	taskReportContent := string(bytes)
	ceTaskID := sonar.GetSonarCETaskID(taskReportContent)
	if ceTaskID == "" {
		s.Logger.Errorf("can not get sonar ce task ID")
		return errors.New("can not get sonar ce task ID")
	}
	analysisID, err := client.WaitForCETaskTobeDone(ceTaskID, time.Minute*10)
	if err != nil {
		s.Logger.Errorf(err.Error())
		return err
	}
	gateInfo, err := client.GetQualityGateInfo(analysisID)
	if err != nil {
		s.Logger.Errorf(err.Error())
		return err
	}
	s.Logger.Infof("Sonar quality gate status: %s", gateInfo.ProjectStatus.Status)
	sonar.VMPrintSonarConditionTables(gateInfo.ProjectStatus.Conditions, s.Logger)
	if gateInfo.ProjectStatus.Status != sonar.QualityGateOK && gateInfo.ProjectStatus.Status != sonar.QualityGateNone {
		return fmt.Errorf("sonar quality gate status was: %s", gateInfo.ProjectStatus.Status)
	}
	return nil
}
