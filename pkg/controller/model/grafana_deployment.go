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
	"os"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-monitoring-grafana-operator/pkg/apis/operator"
	"github.com/IBM/ibm-monitoring-grafana-operator/pkg/apis/operator/v1alpha1"
)

func getPersistentVolume(cr *v1alpha1.Grafana, name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: cr.Spec.PersistentVolume.ClaimName,
				ReadOnly:  true,
			},
		},
	}

}

func getVolumes(cr *v1alpha1.Grafana) []corev1.Volume {
	var volumes []corev1.Volume

	// Volume to store the logs
	volumes = append(volumes,
		corev1.Volume{
			Name: GrafanaLogVolumes,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: GrafanaDatasourceName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: GrafanaPlugins,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: "dashboard-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)

	if cr.Spec.PersistentVolume != nil && cr.Spec.PersistentVolume.Enabled {
		storageVol := getPersistentVolume(cr, "grafana-storage")
		volumes = append(volumes, storageVol)
	} else {
		volumes = append(volumes, corev1.Volume{
			Name: "grafana-storage",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// configmap name also the volume name
	volumes = append(volumes, createVolumeFromCM(GrafanaConfigName),
		createVolumeFromCM(dsConfig),
		createVolumeFromCM(grafanaDBConfig),
		createVolumeFromCM(grafanaDefaultDashboard),
		createVolumeFromCM(grafanaCRD),
		createVolumeFromCM(routerConfig),
		createVolumeFromCM(routerEntry),
		createVolumeFromCM(grafanaLua),
		createVolumeFromCM(utilLua),
	)

	var cert, clientCert string
	if cr.Spec.TLSSecretName != "" && cr.Spec.TLSClientSecretName != "" {
		cert = cr.Spec.TLSSecretName
		clientCert = cr.Spec.TLSClientSecretName
	} else {
		cert = "ibm-monitoring-certs"
		clientCert = "ibm-monitoring-certs"
	}

	volumes = append(volumes, createVolumeFromSecret(cert, "ibm-monitoring-ca-certs"),
		createVolumeFromSecret(cert, "ibm-monitoring-certs"),
		createVolumeFromSecret(clientCert, "ibm-monitoring-client-certs"),
	)

	if DatasourceType(cr) != operator.DSTypeCommonService {
		volumes = append(volumes, createVolumeFromSecret(DSProxyConfigSecName, DSProxyConfigSecName))

	}

	return volumes
}

func getVolumeMounts() []corev1.VolumeMount {
	var mounts []corev1.VolumeMount

	mounts = append(mounts,
		corev1.VolumeMount{
			Name:      GrafanaConfigName,
			MountPath: "/etc/grafana/grafana.ini",
			SubPath:   "grafana.ini",
		},
		corev1.VolumeMount{
			Name:      "dashboard-volume",
			MountPath: "/etc/grafana/dashboards/grafana",
		},
		corev1.VolumeMount{
			Name:      grafanaDBConfig,
			MountPath: "/etc/grafana/provisioning/dashboards",
		},
		corev1.VolumeMount{
			Name:      GrafanaDataVolumes,
			MountPath: "/var/lib/grafana",
		},
		corev1.VolumeMount{
			Name:      GrafanaLogVolumes,
			MountPath: "/var/log/grafana",
		},
		corev1.VolumeMount{
			Name:      GrafanaDatasourceName,
			MountPath: "/etc/grafana/provisioning/datasources",
		},
		corev1.VolumeMount{
			Name:      "ibm-monitoring-certs",
			MountPath: "/opt/ibm/monitoring/certs",
		},
	)

	return mounts
}

func getProbe(delay, timeout, failure int32) *corev1.Probe {

	var port int = 8443
	var scheme corev1.URIScheme = "HTTPS"
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   GrafanaHealthEndpoint,
				Port:   intstr.FromInt(port),
				Scheme: scheme,
			},
		},
		InitialDelaySeconds: delay,
		TimeoutSeconds:      timeout,
		FailureThreshold:    failure,
	}
}

func getContainers(cr *v1alpha1.Grafana) []corev1.Container {

	var resources corev1.ResourceRequirements
	containers := []corev1.Container{}
	image := imageName(os.Getenv(grafanaImageEnv), cr.Spec.BaseImage)
	if cr.Spec.GrafanaConfig != nil && cr.Spec.GrafanaConfig.Resources != nil {
		resources = *cr.Spec.GrafanaConfig.Resources
	} else {
		resources = getContainerResource(cr, "Grafana")
	}

	containers = append(containers,
		corev1.Container{
			Name:  "grafana",
			Image: image,
			Ports: []corev1.ContainerPort{
				{
					Name:          "web",
					ContainerPort: DefaultGrafanaPort,
					Protocol:      "TCP",
				},
			},
			SecurityContext:          getGrafanaSC(),
			Resources:                resources,
			VolumeMounts:             getVolumeMounts(),
			LivenessProbe:            getProbe(40, 35, 15),
			ReadinessProbe:           getProbe(30, 30, 10),
			TerminationMessagePath:   "/dev/termination-log",
			TerminationMessagePolicy: "File",
			ImagePullPolicy:          "IfNotPresent",
		},
		createRouterContainer(cr),
		createDashboardContainer(cr),
	)
	if DatasourceType(cr) != operator.DSTypeCommonService {
		containers = append(containers, *dsProxyContainer(cr))
	}

	return containers
}

func getPodLabels(cr *v1alpha1.Grafana) map[string]string {

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

func getPodAnnotations(cr *v1alpha1.Grafana) map[string]string {

	annotations := map[string]string{
		//"scheduler.alpha.kubernetes.io/critical-pod": "",
		"clusterhealth.ibm.com/dependencies": "cert-manager, auth-idp, icp-management-ingress, common-web-ui, platform-header",
		"productName":                        "IBM Cloud Platform Common Services",
		"productID":                          "068a62892a1e4db39641342e592daa25",
		"productMetric":                      "FREE",
	}
	if cr.Spec.Service != nil && cr.Spec.Service.Annotations != nil {
		mergeMaps(annotations, cr.Spec.Service.Annotations)
	}

	return annotations
}

// hardcode the setting
func getGrafanaSC() *corev1.SecurityContext {
	True := true
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
			Add: []corev1.Capability{"CHOWN", "NET_ADMIN",
				"NET_RAW", "LEASE",
				"SETGID", "SETUID"},
		},
		Privileged:               &True,
		AllowPrivilegeEscalation: &True,
	}

}

func getImagePullSecrets(cr *v1alpha1.Grafana) []corev1.LocalObjectReference {

	secrets := []corev1.LocalObjectReference{}
	if cr.Spec.ImagePullSecrets != nil {
		for _, secret := range cr.Spec.ImagePullSecrets {
			secrets = append(secrets, corev1.LocalObjectReference{
				Name: secret,
			})
		}
	}
	return secrets
}

func getInitContainers(cr *v1alpha1.Grafana) []corev1.Container {

	image := imageName(os.Getenv(routerImageEnv), cr.Spec.InitImage)

	False := false

	volumeMounts := []corev1.VolumeMount{}
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      "grafana-storage",
			MountPath: "/var/lib/grafana",
		},
		corev1.VolumeMount{
			Name:      dsConfig,
			MountPath: "/opt/entry",
		},
		corev1.VolumeMount{
			Name:      GrafanaDatasourceName,
			MountPath: "/etc/grafana/provisioning/datasources",
		},
		corev1.VolumeMount{
			Name:      "ibm-monitoring-ca-certs",
			MountPath: "/opt/ibm/monitoring/ca-certs",
		},
		corev1.VolumeMount{
			Name:      "ibm-monitoring-client-certs",
			MountPath: "/opt/ibm/monitoring/certs",
		},
		corev1.VolumeMount{
			Name:      GrafanaPlugins,
			MountPath: "/var/lib/grafana/plugins",
		},
	)

	SC := corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		AllowPrivilegeEscalation: &False,
		Privileged:               &False,
	}

	return []corev1.Container{
		{
			Name:            InitContainerName,
			Image:           image,
			Command:         []string{"/opt/entry/entrypoint.sh"},
			Resources:       corev1.ResourceRequirements{},
			SecurityContext: &SC,
			VolumeMounts:    volumeMounts,
			ImagePullPolicy: "IfNotPresent",
		},
	}
}

func getDeploymentSpec(cr *v1alpha1.Grafana) appv1.DeploymentSpec {

	selectors := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app":       "grafana",
			"component": "grafana",
		},
	}

	var serviceAccount string
	if cr.Spec.ServiceAccount != "" {
		serviceAccount = cr.Spec.ServiceAccount
	} else {
		serviceAccount = GrafanaServiceAccountName
	}

	// Do not support multiple instance now for 1st release.
	var replicas int32 = 1
	return appv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &selectors,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GrafanaDeploymentName,
				Labels:      getPodLabels(cr),
				Annotations: getPodAnnotations(cr),
			},
			Spec: corev1.PodSpec{
				//PriorityClassName:  "system-cluster-critical",
				ImagePullSecrets:   getImagePullSecrets(cr),
				InitContainers:     getInitContainers(cr),
				HostPID:            false,
				HostIPC:            false,
				HostNetwork:        false,
				Volumes:            getVolumes(cr),
				Containers:         getContainers(cr),
				ServiceAccountName: serviceAccount,
				NodeSelector:       cr.Spec.NodeSelector,
			},
		},
	}
}

func GrafanaDeployment(cr *v1alpha1.Grafana) *appv1.Deployment {
	return &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GrafanaDeploymentName,
			Namespace: cr.Namespace,
		},
		Spec: getDeploymentSpec(cr),
	}
}

func GrafanaDeploymentSelector(cr *v1alpha1.Grafana) client.ObjectKey {

	return client.ObjectKey{
		Name:      GrafanaDeploymentName,
		Namespace: cr.ObjectMeta.Namespace,
	}
}

func ReconciledGrafanaDeployment(cr *v1alpha1.Grafana, current *appv1.Deployment) *appv1.Deployment {
	reconciled := current.DeepCopy()
	spec := getDeploymentSpec(cr)
	reconciled.Spec = spec

	return reconciled
}
