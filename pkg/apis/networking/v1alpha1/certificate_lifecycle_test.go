/*
Copyright 2019 The Knative Authors

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
	"testing"

	"github.com/google/go-cmp/cmp"
	"knative.dev/pkg/apis/duck"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	apitest "knative.dev/pkg/apis/testing"
	corev1 "k8s.io/api/core/v1"
)

func TestCertificateDuckTypes(t *testing.T) {
	tests := []struct {
		name string
		t    duck.Implementable
	}{{
		name: "conditions",
		t:    &duckv1beta1.Conditions{},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := duck.VerifyType(&Certificate{}, test.t)
			if err != nil {
				t.Errorf("VerifyType(Certificate, %T) = %v", test.t, err)
			}
		})
	}
}

func TestCertificateGetGroupVersionKind(t *testing.T) {
	c := Certificate{}
	expected := SchemeGroupVersion.WithKind("Certificate")
	if diff := cmp.Diff(expected, c.GetGroupVersionKind()); diff != "" {
		t.Errorf("Unexpected diff (-want, +got) = %s", diff)
	}
}

func TestMarkReady(t *testing.T) {
	c := &CertificateStatus{}
	c.InitializeConditions()
	apitest.CheckConditionOngoing(c.duck(), CertificateConditionReady, t)

	c.MarkReady()
	if !c.IsReady() {
		t.Error("IsReady=false, want: true")
	}
}

func TestMarkUnknown(t *testing.T) {
	c := &CertificateStatus{}
	c.InitializeConditions()
	apitest.CheckCondition(c.duck(), CertificateConditionReady, corev1.ConditionUnknown)

	c.MarkUnknown("unknow", "unknown")
	apitest.CheckCondition(c.duck(), CertificateConditionReady, corev1.ConditionUnknown)
}

func TestMarkNotReady(t *testing.T) {
	c := &CertificateStatus{}
	c.InitializeConditions()
	apitest.CheckCondition(c.duck(), CertificateConditionReady, corev1.ConditionUnknown)

	c.MarkNotReady("not ready", "not ready")
	apitest.CheckConditionFailed(c.duck(), CertificateConditionReady, t)
}
