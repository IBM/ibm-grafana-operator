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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-monitoring-grafana-operator/pkg/apis/operator/v1alpha1"
)

func getServiceLabels(cr *v1alpha1.Grafana) map[string]string {
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

func getServiceAnnotations(cr *v1alpha1.Grafana) map[string]string {
	if cr.Spec.Service == nil {
		return nil
	}

	return cr.Spec.Service.Annotations
}

func getServiceType(cr *v1alpha1.Grafana) corev1.ServiceType {
	if cr.Spec.Service == nil {
		return corev1.ServiceTypeClusterIP
	}
	if cr.Spec.Service.Type == "" {
		return corev1.ServiceTypeClusterIP
	}
	return cr.Spec.Service.Type
}

func getServicePorts(cr *v1alpha1.Grafana, currentState *corev1.Service) []corev1.ServicePort {
	intPort := DefaultGrafanaPort

	defaultPorts := []corev1.ServicePort{
		{
			Name:     GrafanaHTTPPortName,
			Protocol: "TCP",
			Port:     intPort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 8445,
			},
		},
	}

	if cr.Spec.Service == nil {
		return defaultPorts
	}

	// Re-assign existing node port
	if cr.Spec.Service != nil &&
		currentState != nil &&
		cr.Spec.Service.Type == corev1.ServiceTypeNodePort {
		for _, port := range currentState.Spec.Ports {
			if port.Name == GrafanaHTTPPortName {
				defaultPorts[0].NodePort = port.NodePort
			}
		}
	}

	if cr.Spec.Service.Ports == nil {
		return defaultPorts
	}

	// Don't allow overriding the default port but allow adding
	// additional ports
	for _, port := range cr.Spec.Service.Ports {
		if port.Name == GrafanaHTTPPortName || port.Port == intPort {
			continue
		}
		defaultPorts = append(defaultPorts, port)
	}

	return defaultPorts
}

func getGrafanaSelectors(cr *v1alpha1.Grafana) map[string]string {
	selectors := map[string]string{
		"app":       "grafana",
		"component": "grafana",
	}

	if cr.Spec.Service != nil && cr.Spec.Service.Selector != nil {
		mergeMaps(selectors, cr.Spec.Service.Selector)
	}

	return selectors
}

func GrafanaService(cr *v1alpha1.Grafana) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GrafanaServiceName,
			Namespace:   cr.Namespace,
			Labels:      getServiceLabels(cr),
			Annotations: getServiceAnnotations(cr),
		},
		Spec: corev1.ServiceSpec{
			Ports:     getServicePorts(cr, nil),
			Selector:  getGrafanaSelectors(cr),
			ClusterIP: "",
			Type:      getServiceType(cr),
		},
	}
}

func ReconciledGrafanaService(cr *v1alpha1.Grafana, current *corev1.Service) *corev1.Service {
	reconciled := current.DeepCopy()
	reconciled.Labels = getServiceLabels(cr)
	reconciled.Annotations = getServiceAnnotations(cr)
	reconciled.Spec.Ports = getServicePorts(cr, current)
	reconciled.Spec.Type = getServiceType(cr)
	reconciled.Spec.Selector = getGrafanaSelectors(cr)
	return reconciled
}

func GrafanaServiceSelector(cr *v1alpha1.Grafana) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      GrafanaServiceName,
	}
}
