/*
Copyright 2018 The Knative Authors

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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"knative.dev/pkg/ptr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/knative/serving/pkg/apis/config"
	"github.com/knative/serving/pkg/apis/serving/v1beta1"
)

var (
	defaultResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	ignoreUnexportedResources = cmpopts.IgnoreUnexported(resource.Quantity{})
)

func TestConfigurationDefaulting(t *testing.T) {
	tests := []struct {
		name string
		in   *Configuration
		want *Configuration
		wc   func(context.Context) context.Context
	}{{
		name: "empty",
		in:   &Configuration{},
		want: &Configuration{},
	}, {
		name: "shell",
		in: &Configuration{
			Spec: ConfigurationSpec{
				DeprecatedRevisionTemplate: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						DeprecatedContainer: &corev1.Container{},
					},
				},
			},
		},
		want: &Configuration{
			Spec: ConfigurationSpec{
				DeprecatedRevisionTemplate: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						RevisionSpec: v1beta1.RevisionSpec{
							TimeoutSeconds: ptr.Int64(config.DefaultRevisionTimeoutSeconds),
						},
						DeprecatedContainer: &corev1.Container{
							Name:      config.DefaultUserContainerName,
							Resources: defaultResources,
						},
					},
				},
			},
		},
	}, {
		name: "lemonade",
		wc:   v1beta1.WithUpgradeViaDefaulting,
		in: &Configuration{
			Spec: ConfigurationSpec{
				DeprecatedRevisionTemplate: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						DeprecatedContainer: &corev1.Container{
							Image: "busybox",
						},
					},
				},
			},
		},
		want: &Configuration{
			Spec: ConfigurationSpec{
				Template: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						RevisionSpec: v1beta1.RevisionSpec{
							TimeoutSeconds: ptr.Int64(config.DefaultRevisionTimeoutSeconds),
							PodSpec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:      config.DefaultUserContainerName,
									Image:     "busybox",
									Resources: defaultResources,
								}},
							},
						},
					},
				},
			},
		},
	}, {
		name: "shell podspec",
		in: &Configuration{
			Spec: ConfigurationSpec{
				DeprecatedRevisionTemplate: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						RevisionSpec: v1beta1.RevisionSpec{
							PodSpec: corev1.PodSpec{
								Containers: []corev1.Container{{}},
							},
						},
					},
				},
			},
		},
		want: &Configuration{
			Spec: ConfigurationSpec{
				DeprecatedRevisionTemplate: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						RevisionSpec: v1beta1.RevisionSpec{
							TimeoutSeconds: ptr.Int64(config.DefaultRevisionTimeoutSeconds),
							PodSpec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:      config.DefaultUserContainerName,
									Resources: defaultResources,
								}},
							},
						},
					},
				},
			},
		},
	}, {
		name: "no overwrite values",
		in: &Configuration{
			Spec: ConfigurationSpec{
				DeprecatedRevisionTemplate: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						RevisionSpec: v1beta1.RevisionSpec{
							ContainerConcurrency: 1,
							TimeoutSeconds:       ptr.Int64(99),
						},
						DeprecatedContainer: &corev1.Container{
							Resources: defaultResources,
						},
					},
				},
			},
		},
		want: &Configuration{
			Spec: ConfigurationSpec{
				DeprecatedRevisionTemplate: &RevisionTemplateSpec{
					Spec: RevisionSpec{
						RevisionSpec: v1beta1.RevisionSpec{
							ContainerConcurrency: 1,
							TimeoutSeconds:       ptr.Int64(99),
						},
						DeprecatedContainer: &corev1.Container{
							Name:      config.DefaultUserContainerName,
							Resources: defaultResources,
						},
					},
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.in
			ctx := context.Background()
			if test.wc != nil {
				ctx = test.wc(ctx)
			}
			got.SetDefaults(ctx)
			if diff := cmp.Diff(test.want, got, ignoreUnexportedResources); diff != "" {
				t.Errorf("SetDefaults (-want, +got) = %v", diff)
			}
		})
	}
}
