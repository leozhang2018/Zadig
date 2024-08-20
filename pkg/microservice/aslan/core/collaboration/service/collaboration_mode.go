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

package service

import (
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/collaboration/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/collaboration/repository/mongodb"
)

func validateMemberInfo(collaborationMode *models.CollaborationMode) bool {
	if len(collaborationMode.Members) != len(collaborationMode.MemberInfo) {
		return false
	}
	memberSet := sets.NewString(collaborationMode.Members...)
	memberInfoSet := sets.NewString()
	for _, memberInfo := range collaborationMode.MemberInfo {
		memberInfoSet.Insert(memberInfo.GetID())
	}
	return memberSet.Equal(memberInfoSet)
}

func CreateCollaborationMode(userName string, collaborationMode *models.CollaborationMode, logger *zap.SugaredLogger) error {
	if !validateMemberInfo(collaborationMode) {
		return fmt.Errorf("members and member_info not match")
	}
	err := mongodb.NewCollaborationModeColl().Create(userName, collaborationMode)
	if err != nil {
		logger.Errorf("CreateCollaborationMode error, err msg:%s", err)
		return err
	}
	return nil
}

func UpdateCollaborationMode(userName string, collaborationMode *models.CollaborationMode, logger *zap.SugaredLogger) error {
	if !validateMemberInfo(collaborationMode) {
		return fmt.Errorf("members and member_info not match")
	}
	err := mongodb.NewCollaborationModeColl().Update(userName, collaborationMode)
	if err != nil {
		logger.Errorf("UpdateCollaborationMode error, err msg:%s", err)
		return err
	}
	return nil
}

func DeleteCollaborationMode(username, projectName, name string, logger *zap.SugaredLogger) error {
	err := mongodb.NewCollaborationModeColl().Delete(username, projectName, name)
	if err != nil {
		logger.Errorf("UpdateCollaborationMode error, err msg:%s", err)
		return err
	}
	return mongodb.NewCollaborationInstanceColl().ResetRevision(name, projectName)
}

func GetCollaborationMode(username, projectName, name string, logger *zap.SugaredLogger) (*models.CollaborationMode, bool, error) {
	opt := &mongodb.CollaborationModeFindOptions{
		ProjectName: projectName,
		Name:        name,
	}
	resp, err := mongodb.NewCollaborationModeColl().Find(opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false, nil
		}

		logger.Errorf("UpdateCollaborationMode error, err msg:%s", err)
		return nil, false, err
	}
	return resp, true, nil
}
