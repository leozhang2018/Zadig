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

package util

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	gotemplate "text/template"

	"gopkg.in/yaml.v2"

	commomtemplate "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/service/template"
	"github.com/koderover/zadig/v2/pkg/setting"
)

var (
	templateErrKeyExtractRegex = regexp.MustCompile("<\\.(\\w*)>")
)

func ImproveTemplateExecuteErrReadability(err error) error {
	allKeyMatches := templateErrKeyExtractRegex.FindAllStringSubmatch(err.Error(), -1)
	if allKeyMatches != nil {
		missingKeys := []string{}
		for _, match := range allKeyMatches {
			if len(match) != 2 {
				continue
			}
			missingKeys = append(missingKeys, match[1])
		}
		return fmt.Errorf("template validate err: missing keys %v", missingKeys)
	}
	return fmt.Errorf("template validate err: %w", err)
}

// @fixme MAY NOT support multi variableYamls, need to check
// won't return error if template key is missing values
func RenderK8sSvcYaml(originYaml, productName, serviceName string, variableYamls ...string) (string, error) {
	return renderK8sSvcYamlImpl(originYaml, productName, serviceName, "", variableYamls...)
}

// @fixme MAY NOT support multi variableYamls, need to check
// will return error if template key is missing values
func RenderK8sSvcYamlStrict(originYaml, productName, serviceName string, variableYamls ...string) (string, error) {
	return renderK8sSvcYamlImpl(originYaml, productName, serviceName, "missingkey=error", variableYamls...)
}

func renderK8sSvcYamlImpl(originYaml, productName, serviceName, templateOption string, variableYamls ...string) (string, error) {
	tmpl, err := gotemplate.New(serviceName).Parse(originYaml)
	if err != nil {
		return originYaml, fmt.Errorf("failed to build template, err: %s", err)
	}
	if templateOption != "" {
		tmpl.Option(templateOption)
	}

	variableYaml, replacedKv, err := commomtemplate.SafeMergeVariableYaml(variableYamls...)
	if err != nil {
		return originYaml, err
	}

	variableYaml = strings.ReplaceAll(variableYaml, setting.TemplateVariableProduct, productName)
	variableYaml = strings.ReplaceAll(variableYaml, setting.TemplateVariableService, serviceName)

	variableMap := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(variableYaml), &variableMap)
	if err != nil {
		return originYaml, fmt.Errorf("failed to unmarshal variable yaml, err: %s", err)
	}

	buf := bytes.NewBufferString("")
	err = tmpl.Execute(buf, variableMap)
	if err != nil {
		return originYaml, ImproveTemplateExecuteErrReadability(err)
	}

	originYaml = buf.String()

	// replace system variables
	originYaml = strings.ReplaceAll(originYaml, setting.TemplateVariableProduct, productName)
	originYaml = strings.ReplaceAll(originYaml, setting.TemplateVariableService, serviceName)

	for rk, rv := range replacedKv {
		originYaml = strings.ReplaceAll(originYaml, rk, rv)
	}

	return originYaml, nil
}
