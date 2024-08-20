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

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	"github.com/koderover/zadig/v2/pkg/tool/blueking"
)

func ListBlueKingBusiness(toolID string, page, perPage int64, log *zap.SugaredLogger) (*blueking.BusinessList, error) {
	info, err := mongodb.NewCICDToolColl().Get(toolID)
	if err != nil {
		log.Infof("failed to get tool information of id: %s from mongodb, error: %s", toolID, err)
		return nil, err
	}

	bkClient := blueking.NewClient(info.Host, info.AppCode, info.AppSecret, info.BKUserName)

	start := (page - 1) * perPage
	return bkClient.SearchBusiness(start, perPage)
}

type ListBlueKingExecutionPlanReq struct {
	Total          int                            `json:"total"`
	ExecutionPlans []*blueking.ExecutionPlanBrief `json:"execution_plans"`
}

func ListBlueKingExecutionPlan(toolID string, businessID int64, name string, page, perPage int64, log *zap.SugaredLogger) (*ListBlueKingExecutionPlanReq, error) {
	info, err := mongodb.NewCICDToolColl().Get(toolID)
	if err != nil {
		log.Infof("failed to get tool information of id: %s from mongodb, error: %s", toolID, err)
		return nil, err
	}

	bkClient := blueking.NewClient(info.Host, info.AppCode, info.AppSecret, info.BKUserName)

	start := (page - 1) * perPage
	executionPlanList, err := bkClient.ListExecutionPlan(businessID, name, start, perPage)
	if err != nil {
		return nil, err
	}

	return &ListBlueKingExecutionPlanReq{
		Total:          len(executionPlanList),
		ExecutionPlans: executionPlanList,
	}, nil
}

func GetBlueKingExecutionPlanDetail(toolID string, businessID, planID int64, log *zap.SugaredLogger) (*blueking.ExecutionPlanDetail, error) {
	info, err := mongodb.NewCICDToolColl().Get(toolID)
	if err != nil {
		log.Infof("failed to get tool information of id: %s from mongodb, error: %s", toolID, err)
		return nil, err
	}

	bkClient := blueking.NewClient(info.Host, info.AppCode, info.AppSecret, info.BKUserName)

	return bkClient.GetExecutionPlanDetail(businessID, planID)
}

func GetBlueKingBusinessTopology(toolID string, businessID int64, log *zap.SugaredLogger) ([]*blueking.TopologyNode, error) {
	info, err := mongodb.NewCICDToolColl().Get(toolID)
	if err != nil {
		log.Infof("failed to get tool information of id: %s from mongodb, error: %s", toolID, err)
		return nil, err
	}

	bkClient := blueking.NewClient(info.Host, info.AppCode, info.AppSecret, info.BKUserName)

	return bkClient.GetTopology(businessID)
}

func ListServerByBlueKingTopologyNode(toolID string, businessID, instanceID int64, objectID string, page, perPage int64, log *zap.SugaredLogger) (*blueking.HostList, error) {
	info, err := mongodb.NewCICDToolColl().Get(toolID)
	if err != nil {
		log.Infof("failed to get tool information of id: %s from mongodb, error: %s", toolID, err)
		return nil, err
	}

	bkClient := blueking.NewClient(info.Host, info.AppCode, info.AppSecret, info.BKUserName)

	start := (page - 1) * perPage
	return bkClient.GetHostByTopologyNode(businessID, instanceID, start, perPage, objectID)
}

func ListServerByBlueKingBusiness(toolID string, businessID, page, perPage int64, log *zap.SugaredLogger) (*blueking.HostList, error) {
	info, err := mongodb.NewCICDToolColl().Get(toolID)
	if err != nil {
		log.Infof("failed to get tool information of id: %s from mongodb, error: %s", toolID, err)
		return nil, err
	}

	bkClient := blueking.NewClient(info.Host, info.AppCode, info.AppSecret, info.BKUserName)

	start := (page - 1) * perPage
	return bkClient.GetHostByBusiness(businessID, start, perPage)
}
