//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package model

import (
	ingressv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-monitoring-grafana-operator/pkg/apis/operator/v1alpha1"
)

var GrafanaIngressName string = "grafana-ingress"

func GetIngressLabels(cr *v1alpha1.Grafana) map[string]string {

	labels := map[string]string{
		"app":       "grafana",
		"component": "grafana",
	}
	labels = appendCommonLabels(labels)
	if cr.Spec.Service != nil && cr.Spec.Service.Labels != nil {
		mergeMaps(labels, cr.Spec.Service.Labels)
	}
	return labels
}

func GetIngressAnnotations(cr *v1alpha1.Grafana) map[string]string {
	annotations := map[string]string{
		"kubernetes.io/ingress.class":                    "ibm-icp-management",
		"icp.management.ibm.com/authz-type":              "rbac",
		"icp.management.ibm.com/secure-backends":         "true",
		"icp.management.ibm.com/secure-client-ca-secret": cr.Spec.TLSClientSecretName,
		"icp.management.ibm.com/rewrite-target":          "/",
	}

	if cr.Spec.Service.Annotations != nil && len(cr.Spec.Service.Annotations) != 0 {
		mergeMaps(annotations, cr.Spec.Service.Annotations)
	}
	return annotations
}

func getIngressSpec() ingressv1.IngressSpec {
	pathType := ingressv1.PathType("ImplementationSpecific")
	return ingressv1.IngressSpec{
		Rules: []ingressv1.IngressRule{
			{
				IngressRuleValue: ingressv1.IngressRuleValue{
					HTTP: &ingressv1.HTTPIngressRuleValue{
						Paths: []ingressv1.HTTPIngressPath{
							{
								Path:     "/grafana",
								PathType: &pathType,
								Backend: ingressv1.IngressBackend{
									Service: &ingressv1.IngressServiceBackend{
										Name: GrafanaServiceName,
										Port: ingressv1.ServiceBackendPort{
											Number: DefaultGrafanaPort,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func GrafanaIngress(cr *v1alpha1.Grafana) *ingressv1.Ingress {
	return &ingressv1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Name:        GrafanaIngressName,
			Namespace:   cr.Namespace,
			Labels:      GetIngressLabels(cr),
			Annotations: GetIngressAnnotations(cr),
		},
		Spec: getIngressSpec(),
	}
}

func ReconciledGrafanaIngress(cr *v1alpha1.Grafana, current *ingressv1.Ingress) *ingressv1.Ingress {

	reconciled := current.DeepCopy()
	spec := getIngressSpec()
	reconciled.Spec = spec
	reconciled.Labels = GetIngressLabels(cr)
	reconciled.Annotations = GetIngressAnnotations(cr)
	return reconciled
}

func GrafanaIngressSelector(cr *v1alpha1.Grafana) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      GrafanaIngressName,
	}
}
