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

package service

import (
	"time"

	"go.uber.org/zap"

	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service/kube"
	cluster "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/multicluster/service"
	"github.com/koderover/zadig/v2/pkg/tool/log"
)

func OpenAPICreateRegistry(username string, req *OpenAPICreateRegistryReq, logger *zap.SugaredLogger) error {
	reg := &commonmodels.RegistryNamespace{
		RegAddr:     req.Address,
		RegProvider: string(req.Provider),
		IsDefault:   req.IsDefault,
		Namespace:   req.Namespace,
		AccessKey:   req.AccessKey,
		SecretKey:   req.SecretKey,
		Region:      req.Region,
		UpdateTime:  time.Now().Unix(),
		UpdateBy:    username,
		AdvancedSetting: &commonmodels.RegistryAdvancedSetting{
			Modified:   true,
			TLSEnabled: req.EnableTLS,
			TLSCert:    req.TLSCert,
		},
	}

	return CreateRegistryNamespace(username, reg, logger)
}

func getProjectNames(clusterID string, logger *zap.SugaredLogger) (projectNames []string) {
	projectClusterRelations, err := commonrepo.NewProjectClusterRelationColl().List(&commonrepo.ProjectClusterRelationOption{ClusterID: clusterID})
	if err != nil {
		logger.Errorf("Failed to list projectClusterRelation, err:%s", err)
		return []string{}
	}
	for _, projectClusterRelation := range projectClusterRelations {
		projectNames = append(projectNames, projectClusterRelation.ProjectName)
	}
	return projectNames
}

func OpenAPICreateCluster(projectName string, logger *zap.SugaredLogger) ([]*OpenAPICluster, error) {
	clusters, err := cluster.ListClusters([]string{}, projectName, logger)
	if err != nil {
		logger.Errorf("OpenAPI:ListClusters err : %v", err)
		return nil, err
	}

	resp := make([]*OpenAPICluster, 0)
	for _, cl := range clusters {
		resp = append(resp, &OpenAPICluster{
			ID:           cl.ID,
			Name:         cl.Name,
			Production:   cl.Production,
			Description:  cl.Description,
			ProviderName: ClusterProviderValueNames[cl.Provider],
			CreatedBy:    cl.CreatedBy,
			CreatedTime:  cl.CreatedAt,
			Local:        cl.Local,
			Status:       string(cl.Status),
			Type:         cl.Type,
			ProjectNames: getProjectNames(cl.ID, logger),
		})
	}

	return resp, nil
}

func OpenAPIListCluster(projectName string, logger *zap.SugaredLogger) ([]*OpenAPICluster, error) {
	clusters, err := cluster.ListClusters([]string{}, projectName, logger)
	if err != nil {
		logger.Errorf("OpenAPI:ListClusters err : %v", err)
		return nil, err
	}

	resp := make([]*OpenAPICluster, 0)
	for _, cl := range clusters {
		resp = append(resp, &OpenAPICluster{
			ID:           cl.ID,
			Name:         cl.Name,
			Production:   cl.Production,
			Description:  cl.Description,
			Provider:     cl.Provider,
			ProviderName: ClusterProviderValueNames[cl.Provider],
			CreatedBy:    cl.CreatedBy,
			CreatedTime:  cl.CreatedAt,
			Local:        cl.Local,
			Status:       string(cl.Status),
			Type:         cl.Type,
			ProjectNames: getProjectNames(cl.ID, logger),
		})
	}

	return resp, nil
}

func OpenAPIDeleteCluster(userName, clusterID string, logger *zap.SugaredLogger) error {
	return cluster.DeleteCluster(userName, clusterID, logger)
}

func OpenAPIUpdateCluster(userName, clusterID string, clusterInfo *OpenAPICluster, logger *zap.SugaredLogger) error {
	curClusterInfo, err := cluster.GetCluster(clusterID, logger)
	if err != nil {
		return nil
	}

	curClusterInfo.Name = clusterInfo.Name
	curClusterInfo.Description = clusterInfo.Description
	if curClusterInfo.AdvancedConfig == nil {
		curClusterInfo.AdvancedConfig = &commonmodels.AdvancedConfig{}
	}
	curClusterInfo.AdvancedConfig.ProjectNames = clusterInfo.ProjectNames

	clusterSvc, err := kube.NewService("")
	if err != nil {
		return err
	}

	// Delete all projects associated with clusterID
	err = commonrepo.NewProjectClusterRelationColl().Delete(&commonrepo.ProjectClusterRelationOption{ClusterID: clusterID})
	if err != nil {
		logger.Errorf("Failed to delete projectClusterRelation err:%s", err)
	}
	for _, projectName := range clusterInfo.ProjectNames {
		err = commonrepo.NewProjectClusterRelationColl().Create(&commonmodels.ProjectClusterRelation{
			ProjectName: projectName,
			ClusterID:   clusterID,
			CreatedBy:   userName,
		})
		if err != nil {
			logger.Errorf("Failed to create projectClusterRelation err:%s", err)
		}
	}

	// only update basic info of cluster, like name, description etc
	_, err = clusterSvc.UpdateCluster(clusterID, curClusterInfo, logger)
	return err
}

func K8SClusterModelToOpenAPICluster(clusterResp *commonmodels.K8SCluster) *OpenAPICluster {
	return &OpenAPICluster{
		ID:           clusterResp.ID.Hex(),
		Name:         clusterResp.Name,
		Type:         clusterResp.Type,
		ProviderName: ClusterProviderValueNames[clusterResp.Provider],
		Production:   clusterResp.Production,
		Description:  clusterResp.Description,
		ProjectNames: getProjectNames(clusterResp.ID.Hex(), log.SugaredLogger()),
		Local:        clusterResp.Local,
		Status:       string(clusterResp.Status),
		CreatedBy:    clusterResp.CreatedBy,
		CreatedTime:  clusterResp.CreatedAt,
	}
}
