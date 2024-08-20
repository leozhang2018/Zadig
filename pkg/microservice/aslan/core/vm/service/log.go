/*
Copyright 2023 The KodeRover Authors.

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

package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	utilconfig "github.com/koderover/zadig/v2/pkg/config"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	vmmodel "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models/vm"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	"github.com/koderover/zadig/v2/pkg/setting"
	"github.com/koderover/zadig/v2/pkg/tool/cache"
	"github.com/koderover/zadig/v2/pkg/tool/log"
	s3tool "github.com/koderover/zadig/v2/pkg/tool/s3"
	"github.com/koderover/zadig/v2/pkg/util"
)

var VMJobStatus = VMJobStatusMap{}

type VMJobStatusMap struct {
}

func vmJobKey(key string) string {
	return fmt.Sprintf("vm-job-%s", key)
}

func (v *VMJobStatusMap) Exists(key string) bool {
	exists, err := cache.NewRedisCache(utilconfig.RedisCommonCacheTokenDB()).Exists(vmJobKey(key))
	if err != nil {
		log.Errorf("redis check err: %s for key: %s", err, vmJobKey(key))
	}
	return exists
}

func (v *VMJobStatusMap) Set(key string) {
	// use the timeout value of task timeout should be better
	err := cache.NewRedisCache(utilconfig.RedisCommonCacheTokenDB()).SetNX(vmJobKey(key), "1", 24*time.Hour)
	if err != nil {
		log.Errorf("reids set nx err: %s for key: %s", err, vmJobKey(key))
	}
}

func (v *VMJobStatusMap) Delete(key string) {
	cache.NewRedisCache(utilconfig.RedisCommonCacheTokenDB()).Delete(vmJobKey(key))
}

func savaVMJobLog(job *vmmodel.VMJob, logContent string, logger *zap.SugaredLogger) (err error) {
	if job.Status == string(config.StatusRunning) {
		VMJobStatus.Set(job.ID.Hex())
	}

	var file string
	if job != nil && job.LogFile == "" && logContent != "" {
		file, err = util.CreateVMJobLogFile(job.ID.Hex())
		if err != nil {
			return fmt.Errorf("failed to generate tmp file, error: %s", err)
		}
		job.LogFile = file
	} else {
		file = job.LogFile
	}

	if logContent != "" {
		err = util.WriteFile(file, []byte(logContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write log to file, error: %s", err)
		}
	}

	// after the task execution ends, synchronize the logs in the file to s3
	if job.JobFinished() {
		if err = uploadVMJobLog2S3(job); err != nil {
			logger.Errorf("failed to upload job log to s3, project:%s workflow:%s taskID%d error: %s", job.ProjectName, job.WorkflowName, job.TaskID, err)
			return fmt.Errorf("failed to upload job log to s3, project:%s workflow:%s taskID%d error: %s", job.ProjectName, job.WorkflowName, job.TaskID, err)
		}

		time.Sleep(1000 * time.Millisecond)
		VMJobStatus.Delete(job.ID.Hex())
	}
	return
}

func uploadVMJobLog2S3(job *vmmodel.VMJob) error {
	store, err := commonrepo.NewS3StorageColl().FindDefault()
	if err != nil {
		return fmt.Errorf("failed to get default s3 storage: %s", err)
	}

	if job.LogFile != "" {
		if store.Subfolder != "" {
			store.Subfolder = fmt.Sprintf("%s/%s/%d/%s", store.Subfolder, strings.ToLower(job.WorkflowName), job.TaskID, "log")
		} else {
			store.Subfolder = fmt.Sprintf("%s/%d/%s", strings.ToLower(job.WorkflowName), job.TaskID, "log")
		}
		forcedPathStyle := true
		if store.Provider == setting.ProviderSourceAli {
			forcedPathStyle = false
		}
		s3client, err := s3tool.NewClient(store.Endpoint, store.Ak, store.Sk, store.Region, store.Insecure, forcedPathStyle)
		if err != nil {
			return fmt.Errorf("saveContainerLog s3 create client error: %v", err)
		}
		fileName := strings.Replace(strings.ToLower(job.JobName), "_", "-", -1)
		objectKey := GetObjectPath(store.Subfolder, fileName+".log")
		if err = s3client.Upload(
			store.Bucket,
			job.LogFile,
			objectKey,
		); err != nil {
			return fmt.Errorf("saveContainerLog s3 Upload error: %v", err)
		}

		// remove the log file later
		util.Go(func() {
			time.Sleep(5 * time.Second)
			err = os.Remove(job.LogFile)
			if err != nil {
				log.Errorf("Failed to remove vm job log file, error: %v", err)
			}
		})

		log.Infof("saveContainerLog s3 upload success, workflowName:%s jobName:%s, taskID:%d", job.WorkflowName, job.JobName, job.TaskID)
	}

	return nil
}

func GetObjectPath(subFolder, name string) string {
	// target should not be started with /
	if subFolder != "" {
		return strings.TrimLeft(filepath.Join(subFolder, name), "/")
	}

	return strings.TrimLeft(name, "/")
}
