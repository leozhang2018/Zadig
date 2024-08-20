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
	"context"
	"testing"

	"github.com/magiconair/properties/assert"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCronJobSuccess(t *testing.T) {
	clientset := fake.NewSimpleClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-cronjob",
			Namespace: "default",
			Annotations: map[string]string{
				"analysisDate": "2022-04-01",
			},
			Labels: map[string]string{
				"app": "example-app",
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          "*/1 * * * *",
			ConcurrencyPolicy: "Allow",
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "example-app",
					},
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "example-container",
									Image: "nginx",
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	})

	config := Analyzer{
		Client: &Client{
			Client: clientset,
		},
		Context:   context.Background(),
		Namespace: "default",
	}

	analyzer := CronJobAnalyzer{}
	analysisResults, err := analyzer.Analyze(config)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(analysisResults), 0)
}

func TestCronJobBroken(t *testing.T) {
	clientset := fake.NewSimpleClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-cronjob",
			Namespace: "default",
			Annotations: map[string]string{
				"analysisDate": "2022-04-01",
			},
			Labels: map[string]string{
				"app": "example-app",
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          "*** * * * *",
			ConcurrencyPolicy: "Allow",
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "example-app",
					},
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "example-container",
									Image: "nginx",
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	})

	config := Analyzer{
		Client: &Client{
			Client: clientset,
		},
		Context:   context.Background(),
		Namespace: "default",
	}

	analyzer := CronJobAnalyzer{}
	analysisResults, err := analyzer.Analyze(config)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(analysisResults), 1)
	assert.Equal(t, analysisResults[0].Name, "default/example-cronjob")
	assert.Equal(t, analysisResults[0].Kind, "CronJob")
}

func TestCronJobBrokenMultipleNamespaceFiltering(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "example-cronjob",
				Namespace: "default",
				Annotations: map[string]string{
					"analysisDate": "2022-04-01",
				},
				Labels: map[string]string{
					"app": "example-app",
				},
			},
			Spec: batchv1.CronJobSpec{
				Schedule:          "*** * * * *",
				ConcurrencyPolicy: "Allow",
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "example-app",
						},
					},
					Spec: batchv1.JobSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name:  "example-container",
										Image: "nginx",
									},
								},
								RestartPolicy: v1.RestartPolicyOnFailure,
							},
						},
					},
				},
			},
		},
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "example-cronjob",
				Namespace: "other-namespace",
				Annotations: map[string]string{
					"analysisDate": "2022-04-01",
				},
				Labels: map[string]string{
					"app": "example-app",
				},
			},
			Spec: batchv1.CronJobSpec{
				Schedule:          "*** * * * *",
				ConcurrencyPolicy: "Allow",
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "example-app",
						},
					},
					Spec: batchv1.JobSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name:  "example-container",
										Image: "nginx",
									},
								},
								RestartPolicy: v1.RestartPolicyOnFailure,
							},
						},
					},
				},
			},
		})

	config := Analyzer{
		Client: &Client{
			Client: clientset,
		},
		Context:   context.Background(),
		Namespace: "default",
	}

	analyzer := CronJobAnalyzer{}
	analysisResults, err := analyzer.Analyze(config)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(analysisResults), 1)
	assert.Equal(t, analysisResults[0].Name, "default/example-cronjob")
	assert.Equal(t, analysisResults[0].Kind, "CronJob")
}
