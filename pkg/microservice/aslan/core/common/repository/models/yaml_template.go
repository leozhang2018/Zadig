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

package models

import (
	commontypes "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type YamlTemplate struct {
	ID                 primitive.ObjectID               `bson:"_id,omitempty"        json:"id,omitempty"`
	Name               string                           `bson:"name"                 json:"name"`
	Content            string                           `bson:"content"              json:"content"`
	Variables          []*Variable                      `bson:"variables"            json:"variables"` // Deprecated since 1.16.0
	VariableYaml       string                           `bson:"variable_yaml"        json:"variable_yaml"`
	ServiceVariableKVs []*commontypes.ServiceVariableKV `bson:"service_variable_kvs" json:"service_variable_kvs"`
	ServiceVars        []string                         `bson:"service_vars"         json:"service_vars"` // Deprecated since 1.18.0
}

type Variable struct {
	Key   string `bson:"key"   json:"key"`
	Value string `bson:"value" json:"value"`
}

func (YamlTemplate) TableName() string {
	return "yaml_template"
}
