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

package executor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	commonconfig "github.com/koderover/zadig/v2/pkg/config"
	job "github.com/koderover/zadig/v2/pkg/microservice/jobexecutor/core/service"
	"github.com/koderover/zadig/v2/pkg/microservice/jobexecutor/core/service/configmap"
	"github.com/koderover/zadig/v2/pkg/setting"
	"github.com/koderover/zadig/v2/pkg/tool/log"
	"github.com/koderover/zadig/v2/pkg/types"
)

func Execute(ctx context.Context) error {
	log.Init(&log.Config{
		Level:         commonconfig.LogLevel(),
		NoCaller:      true,
		NoLogLevel:    true,
		Development:   commonconfig.Mode() != setting.ReleaseMode,
		WorkflowStyle: true,
		// SendToFile:  true,
		// Filename:    ZadigLogFile,
	})

	start := time.Now()

	excutor := "job-executor"

	var (
		err error
		j   *job.Job
	)
	j, err = job.NewJob()
	if err != nil {
		return err
	}

	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Errorf("Failed to get namespace, err: %v", err)
		return errors.Wrap(err, "get namespace")
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("failed to get InClusterConfig, err: %v", err)
		return errors.Wrap(err, "get InClusterConfig")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("failed to get ClientSet, err: %v", err)
		return errors.Wrap(err, "get ClientSet")
	}

	j.ConfigMapUpdater = configmap.NewUpdater(j.Ctx.ConfigMapName, string(ns), clientset)

	defer func() {
		resultMsg := types.JobSuccess
		if err != nil {
			resultMsg = types.JobFail
			fmt.Printf("Failed to run: %s.\n", err)
		}
		fmt.Printf("%s   Job Status: %s\n", time.Now().Format(setting.WorkflowTimeFormat), resultMsg)

		// set job status and outputs to job context configMap
		cm, err := j.ConfigMapUpdater.Get()
		if err != nil {
			log.Errorf("failed to get job context ConfigMap: %v", err)
			return
		}
		cm.Data[types.JobResultKey] = string(resultMsg)
		cm.Data[types.JobOutputsKey] = string(j.OutputsJsonBytes)
		if j.ConfigMapUpdater.UpdateWithRetry(cm, 3, 3*time.Second) != nil {
			log.Errorf("failed to update job context ConfigMap: %v", err)
			return
		}
		log.Infof("Job result ConfigMap is updated successfully")
		fmt.Printf("====================== %s End. Duration: %.2f seconds ======================\n", excutor, time.Since(start).Seconds())
	}()

	fmt.Printf("====================== %s Start ======================\n", excutor)
	var (
		runErr      error
		afterRunErr error
	)

	// in order to collect output vars, we don't immediately return err even if runErr is not nil
	runErr = j.Run(ctx)
	afterRunErr = j.AfterRun(ctx)
	if runErr != nil {
		if afterRunErr != nil {
			log.Errorf("AfterRun error: %v", afterRunErr)
		}
		err = runErr
		return err
	}
	if afterRunErr != nil {
		err = afterRunErr
		return err
	}

	return nil
}
