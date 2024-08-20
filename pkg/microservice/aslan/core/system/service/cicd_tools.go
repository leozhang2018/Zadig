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

package service

import (
	"go.uber.org/zap"

	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service"
	"github.com/koderover/zadig/v2/pkg/setting"
	"github.com/koderover/zadig/v2/pkg/tool/crypto"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
)

func CreateCICDTools(args *commonmodels.JenkinsIntegration, log *zap.SugaredLogger) error {
	if err := commonrepo.NewCICDToolColl().Create(args); err != nil {
		log.Errorf("Create CI/CD Tools err:%v", err)
		return e.ErrCreateCICDTools.AddErr(err)
	}
	return nil
}

func ListCICDTools(encryptedKey, toolType string, log *zap.SugaredLogger) ([]*commonmodels.JenkinsIntegration, error) {
	tools, err := commonrepo.NewCICDToolColl().List(toolType)
	if err != nil {
		log.Errorf("List CI/CD Tools err:%v", err)
		return []*commonmodels.JenkinsIntegration{}, e.ErrListCICDTools.AddErr(err)
	}

	if len(encryptedKey) == 0 {
		return tools, nil
	}

	aesKey, err := service.GetAesKeyFromEncryptedKey(encryptedKey, log)
	if err != nil {
		log.Errorf("List CI/CD Tools GetAesKeyFromEncryptedKey err:%v", err)
		return nil, err
	}
	for _, tool := range tools {
		if tool.Type == "" {
			// compatibility code, if a type is empty, it must be legacy data, which only have jenkins as ci/cd tools.
			tool.Type = setting.CICDToolTypeJenkins
		}
		if tool.Name == "" {
			// compatibility code, if a name is empty, it must be legacy data, which only have jenkins as ci/cd tools.
			tool.Name = setting.CICDToolTypeJenkins
		}

		if tool.Type == setting.CICDToolTypeJenkins || tool.Type == "" {
			tool.Password, err = crypto.AesEncryptByKey(tool.Password, aesKey.PlainText)
			if err != nil {
				log.Errorf("List CI/CD Tools AesEncryptByKey err:%v", err)
				return nil, err
			}
		} else if tool.Type == setting.CICDToolTypeBlueKing {
			tool.AppSecret, err = crypto.AesEncryptByKey(tool.AppSecret, aesKey.PlainText)
			if err != nil {
				log.Errorf("List CI/CD Tools AesEncryptByKey err:%v", err)
				return nil, err
			}
		}

	}
	return tools, nil
}

func UpdateCICDTools(ID string, args *commonmodels.JenkinsIntegration, log *zap.SugaredLogger) error {
	if err := commonrepo.NewCICDToolColl().Update(ID, args); err != nil {
		log.Errorf("Update CI/CD tools err:%v", err)
		return e.ErrUpdateCICDTools.AddErr(err)
	}
	return nil
}

func DeleteCICDTools(ID string, log *zap.SugaredLogger) error {
	if err := commonrepo.NewCICDToolColl().Delete(ID); err != nil {
		log.Errorf("Delete CI/CD tools err:%v", err)
		return e.ErrDeleteCICDTools.AddErr(err)
	}
	return nil
}
