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
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/koderover/zadig/v2/pkg/config"
	"github.com/koderover/zadig/v2/pkg/microservice/systemconfig/core/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/systemconfig/core/repository/orm"
	"github.com/koderover/zadig/v2/pkg/shared/client/aslan"
	"github.com/koderover/zadig/v2/pkg/tool/crypto"
)

func ListConnectorsInternal(logger *zap.SugaredLogger) ([]*Connector, error) {
	cs, err := orm.NewConnectorColl().List()
	if err != nil {
		logger.Errorf("Failed to list connectors, err: %s", err)
		return nil, err
	}

	var res []*Connector
	for _, c := range cs {
		cf := make(map[string]interface{})
		err = json.Unmarshal([]byte(c.Config), &cf)
		if err != nil {
			logger.Errorf("Failed to unmarshal config, err: %s", err)
			continue
		}
		res = append(res, &Connector{
			ConnectorBase: ConnectorBase{
				Type: ConnectorType(c.Type),
			},
			ID:     c.ID,
			Name:   c.Name,
			Config: cf,
		})
	}

	return res, nil
}

func ListConnectors(encryptedKey string, logger *zap.SugaredLogger) ([]*Connector, error) {
	aesKey, err := aslan.New(config.AslanServiceAddress()).GetTextFromEncryptedKey(encryptedKey)
	if err != nil {
		logger.Errorf("ListConnectors GetTextFromEncryptedKey, err: %s", err)
		return nil, err
	}
	cs, err := orm.NewConnectorColl().List()
	if err != nil {
		logger.Errorf("Failed to list connectors, err: %s", err)
		return nil, err
	}
	config, err := aslan.New(config.AslanServiceAddress()).GetDefaultLogin()
	if err != nil {
		logger.Errorf("Failed to list connectors, err: %s", err)
		return nil, err
	}
	var res []*Connector
	for _, c := range cs {
		cf := make(map[string]interface{})
		err = json.Unmarshal([]byte(c.Config), &cf)
		if err != nil {
			logger.Errorf("Failed to unmarshal config, err: %s", err)
			continue
		}
		if pw, ok := cf["bindPW"]; ok {
			cf["bindPW"], err = crypto.AesEncryptByKey(pw.(string), aesKey.PlainText)
			if err != nil {
				logger.Errorf("ListConnectors AesEncryptByKey, err: %s", err)
				return nil, err
			}
		}
		if clientSecret, ok := cf["clientSecret"]; ok {
			cf["clientSecret"], err = crypto.AesEncryptByKey(clientSecret.(string), aesKey.PlainText)
			if err != nil {
				logger.Errorf("ListConnectors AesEncryptByKey, err: %s", err)
				return nil, err
			}
		}
		isDefault := false
		if config.DefaultLogin == c.ID {
			isDefault = true
		}
		res = append(res, &Connector{
			ConnectorBase: ConnectorBase{
				Type: ConnectorType(c.Type),
			},
			ID:                c.ID,
			Name:              c.Name,
			Config:            cf,
			IsDefault:         isDefault,
			EnableLogOut:      c.EnableLogOut,
			LogoutRedirectURL: c.LogoutRedirectURL,
		})
	}

	return res, nil
}

func GetConnector(id string, logger *zap.SugaredLogger) (*Connector, error) {
	c, err := orm.NewConnectorColl().Get(id)
	if err != nil {
		logger.Errorf("Failed to get connector %s, err: %s", id, err)
		return nil, err
	}

	cf := make(map[string]interface{})
	err = json.Unmarshal([]byte(c.Config), &cf)
	if err != nil {
		logger.Warnf("Failed to unmarshal config, err: %s", err)

	}

	return &Connector{
		ConnectorBase: ConnectorBase{
			Type: ConnectorType(c.Type),
		},
		ID:                c.ID,
		Name:              c.Name,
		Config:            cf,
		EnableLogOut:      c.EnableLogOut,
		LogoutRedirectURL: c.LogoutRedirectURL,
	}, nil

}

func DeleteConnector(id string, _ *zap.SugaredLogger) error {
	return orm.NewConnectorColl().Delete(id)
}

func CreateConnector(ct *Connector, logger *zap.SugaredLogger) error {
	cf, err := json.Marshal(ct.Config)
	if err != nil {
		logger.Errorf("Failed to marshal config, err: %s", err)
		return err
	}

	cfg := make(map[string]interface{})
	err = json.Unmarshal(cf, &cfg)
	if err != nil {
		logger.Errorf("Failed to unmarshal config, err: %s", err)
		return fmt.Errorf("invalid config")
	}

	if string(ct.Type) != "oauth" && ct.EnableLogOut {
		return fmt.Errorf("logout is only available in oauth2 connector")
	}

	obj := &models.Connector{
		ID:                ct.ID,
		Name:              ct.Name,
		Type:              string(ct.Type),
		Config:            string(cf),
		EnableLogOut:      ct.EnableLogOut,
		LogoutRedirectURL: ct.LogoutRedirectURL,
	}

	return orm.NewConnectorColl().Create(obj)
}

func UpdateConnector(ct *Connector, logger *zap.SugaredLogger) error {
	cf, err := json.Marshal(ct.Config)
	if err != nil {
		logger.Errorf("Failed to marshal config, err: %s", err)
		return err
	}
	
	cfg := make(map[string]interface{})
	err = json.Unmarshal(cf, &cfg)
	if err != nil {
		logger.Errorf("Failed to unmarshal config, err: %s", err)
		return fmt.Errorf("invalid config")
	}

	if string(ct.Type) != "oauth" && ct.EnableLogOut {
		return fmt.Errorf("logout is only available in oauth2 connector")
	}

	obj := &models.Connector{
		ID:                ct.ID,
		Name:              ct.Name,
		Type:              string(ct.Type),
		Config:            string(cf),
		EnableLogOut:      ct.EnableLogOut,
		LogoutRedirectURL: ct.LogoutRedirectURL,
	}

	return orm.NewConnectorColl().Update(obj)
}
