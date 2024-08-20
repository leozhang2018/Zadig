/*
Copyright 2023 The K8sGPT Authors.
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

// Some parts of this file have been modified to make it functional in Zadig

package analysis

import (
	"fmt"

	kubernetes "github.com/koderover/zadig/v2/pkg/shared/kube/wrapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type PdbAnalyzer struct{}

func (PdbAnalyzer) Analyze(a Analyzer) ([]Result, error) {

	kind := "PodDisruptionBudget"
	apiDoc := kubernetes.K8sApiReference{
		Kind: kind,
		ApiVersion: schema.GroupVersion{
			Group:   "policy",
			Version: "v1",
		},
		OpenapiSchema: a.OpenapiSchema,
	}

	AnalyzerErrorsMetric.DeletePartialMatch(map[string]string{
		"analyzer_name": kind,
	})

	list, err := a.Client.GetClient().PolicyV1().PodDisruptionBudgets(a.Namespace).List(a.Context, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var preAnalysis = map[string]PreAnalysis{}

	for _, pdb := range list.Items {
		var failures []Failure

		evt, err := FetchLatestEvent(a.Context, a.Client, pdb.Namespace, pdb.Name)
		if err != nil || evt == nil {
			continue
		}

		if evt.Reason == "NoPods" && evt.Message != "" {
			if pdb.Spec.Selector != nil {
				for k, v := range pdb.Spec.Selector.MatchLabels {
					doc := apiDoc.GetApiDocV2("spec.selector.matchLabels")

					failures = append(failures, Failure{
						Text:          fmt.Sprintf("%s, expected label %s=%s", evt.Message, k, v),
						KubernetesDoc: doc,
						Sensitive: []Sensitive{
							{
								Unmasked: k,
								Masked:   MaskString(k),
							},
							{
								Unmasked: v,
								Masked:   MaskString(v),
							},
						},
					})
				}
				for _, v := range pdb.Spec.Selector.MatchExpressions {
					doc := apiDoc.GetApiDocV2("spec.selector.matchExpressions")

					failures = append(failures, Failure{
						Text:          fmt.Sprintf("%s, expected expression %s", evt.Message, v),
						KubernetesDoc: doc,
						Sensitive:     []Sensitive{},
					})
				}
			} else {
				doc := apiDoc.GetApiDocV2("spec.selector")

				failures = append(failures, Failure{
					Text:          fmt.Sprintf("%s, selector is nil", evt.Message),
					KubernetesDoc: doc,
					Sensitive:     []Sensitive{},
				})
			}
		}

		if len(failures) > 0 {
			preAnalysis[fmt.Sprintf("%s/%s", pdb.Namespace, pdb.Name)] = PreAnalysis{
				PodDisruptionBudget: pdb,
				FailureDetails:      failures,
			}
			AnalyzerErrorsMetric.WithLabelValues(kind, pdb.Name, pdb.Namespace).Set(float64(len(failures)))
		}
	}

	for key, value := range preAnalysis {
		var currentAnalysis = Result{
			Kind:  kind,
			Name:  key,
			Error: value.FailureDetails,
		}

		parent, _ := GetParent(a.Client, value.PodDisruptionBudget.ObjectMeta)
		currentAnalysis.ParentObject = parent
		a.Results = append(a.Results, currentAnalysis)
	}

	return a.Results, err
}
