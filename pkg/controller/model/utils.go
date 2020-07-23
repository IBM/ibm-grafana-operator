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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/IBM/ibm-monitoring-grafana-operator/pkg/apis/operator/v1alpha1"
)

var memoryRequest int = 256
var cpuRequest int = 200
var memoryLimit int = 512
var cpuLimit int = 500
var defaultGrafanaResource, defaultDashboardResource, defaultRouterResource corev1.ResourceRequirements

func mergeMaps(to, from map[string]string) {
	for key, val := range from {
		to[key] = val
	}
}
func imageName(defaultV string, overwrite string) string {
	if strings.Contains(overwrite, imageDigestKey) {
		return overwrite
	}

	return defaultV

}

func defaultResource(name string) corev1.ResourceRequirements {

	resource := func(rc, rm, lc, lm string) corev1.ResourceRequirements {
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(rm),
				corev1.ResourceCPU:    resource.MustParse(rc),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(lm),
				corev1.ResourceCPU:    resource.MustParse(lc),
			},
		}
	}

	if name == "Grafana" {
		defaultGrafanaResource = resource("256Mi", "200m", "512Mi", "500m")
		return defaultGrafanaResource
	} else if name == "Dashboard" {
		defaultDashboardResource = resource("64Mi", "50m", "192Mi", "150m")
		return defaultDashboardResource
	} else if name == "Router" {
		defaultRouterResource = resource("64Mi", "50m", "128Mi", "100m")
		return defaultRouterResource
	}

	// Should not go here.
	return corev1.ResourceRequirements{}

}

func getContainerResource(cr *v1alpha1.Grafana, name string) corev1.ResourceRequirements {

	res := map[string]corev1.ResourceRequirements{}
	var resources *v1alpha1.GrafanaResources

	if cr.Spec.Resources != nil {
		resources = cr.Spec.Resources
	} else {
		return defaultResource(name)
	}

	if resources.GrafanaResource != nil {
		res["Grafana"] = *resources.GrafanaResource
	} else if resources.Grafana != 0 {
		res["Grafana"] = getResource(resources.Grafana)
	} else {
		res["Grafana"] = defaultGrafanaResource
	}

	if resources.DashboardResource != nil {
		res["Dashboard"] = *resources.DashboardResource
	} else if resources.Dashboard != 0 {
		res["Dashboard"] = getResource(resources.Dashboard)
	} else {
		res["Dashboard"] = defaultDashboardResource
	}

	if resources.RouterResource != nil {
		res["Router"] = *resources.RouterResource
	} else if resources.Router != 0 {
		res["Router"] = getResource(resources.Router)
	} else {
		res["Router"] = defaultRouterResource
	}

	return res[name]

}

func getResource(times int) corev1.ResourceRequirements {

	MR := strconv.Itoa(memoryRequest*times) + "Mi"
	CR := strconv.Itoa(cpuRequest*times) + "m"
	ML := strconv.Itoa(memoryLimit*times) + "Mi"
	CL := strconv.Itoa(cpuLimit*times) + "m"
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse(MR),
			corev1.ResourceCPU:    resource.MustParse(CR),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse(ML),
			corev1.ResourceCPU:    resource.MustParse(CL),
		},
	}

}

func createVolumeFromCM(name string) corev1.Volume {

	var stringMode string

	stringMode = "0664"
	if strings.Contains(name, "entry") {
		stringMode = "0777"
	}

	mode, _ := strconv.ParseInt(stringMode, 8, 32)
	defaultMode := int32(mode)

	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				DefaultMode: &defaultMode,
			},
		},
	}
}

func createVolumeFromSecret(secretName, volumeName string) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}

func setupAdminEnv(username, password string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: username,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GrafanaAdminSecretName,
					},
					Key: GrafanaAdminUserEnvVar,
				},
			},
		},
		{
			Name: password,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GrafanaAdminSecretName,
					},
					Key: GrafanaAdminPasswordEnvVar,
				},
			},
		},
	}
}

func appendCommonLabels(labels map[string]string) map[string]string {
	labels["app.kubernetes.io/name"] = "ibm-monitoring"
	labels["app.kubernetes.io/instance"] = "common-monitoring"
	labels["app.kubernetes.io/managed-by"] = "ibm-monitoring-exporters-operator"
	return labels
}
