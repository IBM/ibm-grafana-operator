//
// Copyright 2020-2021 IBM Corporation
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
package grafana

import (
	"fmt"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ingressv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/IBM/ibm-monitoring-grafana-operator/pkg/apis/operator/v1alpha1"
	"github.com/IBM/ibm-monitoring-grafana-operator/pkg/controller/dashboards"

	utils "github.com/IBM/ibm-monitoring-grafana-operator/pkg/controller/model"
)

var IsGrafanaRunning bool = false

func reconcileGrafana(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	err := checkApplicationMonitoring(r, cr)
	if err != nil {
		log.Error(err, "Fail to check OCP application monitoring status")
		return err
	}
	err = reconcileAllConfigMaps(r, cr)
	if err != nil {
		log.Error(err, "Fail to reconcile all the confimags.")
		return err
	}

	err = reconcileCert(r, cr)
	if err != nil {
		log.Error(err, "Fail to reconcile certificate")
		return err
	}

	err = reconcileDSProxyConfigSecret(r, cr)
	if err != nil {
		log.Error(err, "Fail to reconcile datasource proxy configration secret.")
		return err
	}

	err = reconcileGrafanaService(r, cr)
	if err != nil {
		log.Error(err, "Fail to reconcile grafana service.")
		return err
	}
	err = reconcileGrafanaIngress(r, cr)
	if err != nil {
		log.Error(err, "Fail to reconcile grafana ingress.")
	}

	err = reconcileGrafanaSecret(r, cr)
	if err != nil {
		log.Error(err, "Fail to reconcile grafana secret.")
	}

	err = reconcileGrafanaDeployment(r, cr)
	if err != nil {
		log.Error(err, "Fail to reconcile grafana deployment.")
		return err
	}

	err = reconcileAllDashboards(r, cr)
	if err != nil {
		log.Error(err, "Fail to  reconcile grafana dashboards.")
		return err
	}

	err = cleanupCSMonitoring(r, cr)
	if err != nil {
		// no need to return error here as its just cleanup and no impact if it fails
		log.Info("Fail to cleanup one or more old CS Monitoring resources.")
		r.recorder.Eventf(cr, corev1.EventTypeNormal, "If some of old CS monitoring resources are not cleanedup", "Refer the doc for manual cleanup")
		return nil
	}

	return nil
}

func reconcileAllConfigMaps(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {
	configmaps := utils.ReconcileConfigMaps(cr)
	selector := func(name string) client.ObjectKey {
		return client.ObjectKey{
			Namespace: cr.Namespace,
			Name:      name,
		}
	}

	create := func(cm *corev1.ConfigMap) error {
		err := controllerutil.SetControllerReference(cr, cm, r.scheme)
		if err != nil {
			return err
		}
		err = r.client.Create(r.ctx, cm)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("configmap %s is created.", cm.ObjectMeta.Name))
		return nil
	}

	update := func(cm *corev1.ConfigMap) error {
		err := controllerutil.SetControllerReference(cr, cm, r.scheme)
		if err != nil {
			return err
		}
		err = r.client.Update(r.ctx, cm)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("configmap %s is updated.", cm.ObjectMeta.Name))
		return nil
	}

	log.Info("Start to reconcile all the confimaps")
	for _, cm := range configmaps {
		name := cm.ObjectMeta.Name
		ocm := corev1.ConfigMap{}
		err := r.client.Get(r.ctx, selector(name), &ocm)
		if err != nil {
			if errors.IsNotFound(err) {
				err = create(cm)
				if err != nil {
					log.Error(err, fmt.Sprintf("Fail to create configmap %s", name))
					return err
				}
				log.Info(fmt.Sprintf("configmap %s created.", name))
			}
			return err
		}

		err = update(cm)
		if err != nil {
			log.Error(err, fmt.Sprintf("Fail to update configmap %s", name))
			return err
		}
	}
	return nil
}

func reconcileAllDashboards(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	var namespace string = cr.Namespace

	log.Info("Start to reconcile grafana dashboards")

	// Update the dashboards status
	if cr.Spec.DashboardsConfig != nil && cr.Spec.DashboardsConfig.MainOrg != "" {
		namespace = cr.Spec.DashboardsConfig.MainOrg
	}

	dashboards.ReconcileDashboardsStatus(cr)

	// Reconcile all the dashboards
	// Could not get the dashboard resource and workaround this.
	for name, status := range dashboards.DefaultDBsStatus {
		db := dashboards.CreateDashboard(namespace, name, status)
		_ = controllerutil.SetControllerReference(cr, db, r.scheme)
		err := r.client.Create(r.ctx, db)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				continue
			} else {
				log.Error(err, "fail to create dashboard", name)
				return err
			}
		}
	}
	return nil
}

func reconcileGrafanaSecret(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	selector := utils.GrafanaSecretSelector(cr)
	secret := utils.CreateGrafanaSecret(cr)
	err := controllerutil.SetControllerReference(cr, secret, r.scheme)
	if err != nil {
		return err
	}
	err = r.client.Get(r.ctx, selector, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			err = r.client.Create(r.ctx, secret)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	err = r.client.Update(r.ctx, secret)
	if err != nil {
		return err
	}
	return nil
}

func reconcileGrafanaDeployment(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	selector := utils.GrafanaDeploymentSelector(cr)
	deployment := &appv1.Deployment{}
	err := r.client.Get(r.ctx, selector, deployment)
	if err != nil && errors.IsNotFound(err) {
		err = createGrafanaDeployment(r, cr)
		if err != nil {
			log.Error(err, "Fail to create grafana deployment.")
			return err
		}
		log.Info("Grafana deployment created")
		return nil
	}
	if err != nil {
		log.Error(err, "Fail to get grafana deployment.")
		return err
	}

	toUpdate := utils.ReconciledGrafanaDeployment(cr, deployment)

	certmanagerLabel := "certmanager.k8s.io/time-restarted"
	// Preserve cert-manager added labels in metadata
	if val, ok := deployment.ObjectMeta.Labels[certmanagerLabel]; ok {
		toUpdate.ObjectMeta.Labels[certmanagerLabel] = val
	}

	// Preserve cert-manager added labels in spec
	if val, ok := deployment.Spec.Template.ObjectMeta.Labels[certmanagerLabel]; ok {
		toUpdate.Spec.Template.ObjectMeta.Labels[certmanagerLabel] = val
	}
	err = r.client.Update(r.ctx, toUpdate)
	if err != nil {
		log.Error(err, "Fail to update grafana deployment.")
		return err
	}
	log.Info("Grafana deployment updated")

	return nil
}

func createGrafanaDeployment(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	dep := utils.GrafanaDeployment(cr)
	err := controllerutil.SetControllerReference(cr, dep, r.scheme)
	if err != nil {
		return err
	}

	err = r.client.Create(r.ctx, dep)
	if err != nil {
		return err
	}

	return nil
}

func reconcileGrafanaService(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	selector := utils.GrafanaServiceSelector(cr)
	svc := &corev1.Service{}
	err := r.client.Get(r.ctx, selector, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			err = createGrafanaService(r, cr)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	toUpdate := utils.ReconciledGrafanaService(cr, svc)
	err = r.client.Update(r.ctx, toUpdate)
	if err != nil {
		return err
	}
	return nil
}

func createGrafanaService(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {
	svc := utils.GrafanaService(cr)
	err := controllerutil.SetControllerReference(cr, svc, r.scheme)

	if err != nil {
		return err
	}

	err = r.client.Create(r.ctx, svc)
	if err != nil {
		return err
	}

	return nil

}

func reconcileGrafanaIngress(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	selector := utils.GrafanaIngressSelector(cr)
	route := &ingressv1.Ingress{}
	err := r.client.Get(r.ctx, selector, route)
	if err != nil {
		if errors.IsNotFound(err) {
			err = createGrafanaIngress(r, cr)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	toUpdate := utils.ReconciledGrafanaIngress(cr, route)

	err = r.client.Update(r.ctx, toUpdate)
	if err != nil {
		return err
	}
	return nil
}

func createGrafanaIngress(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {
	route := utils.GrafanaIngress(cr)
	err := controllerutil.SetControllerReference(cr, route, r.scheme)
	if err != nil {
		return err
	}

	err = r.client.Create(r.ctx, route)
	if err != nil {
		return err
	}
	return nil
}

func handleError(r *ReconcileGrafana, cr *v1alpha1.Grafana, issue error) (reconcile.Result, error) {
	cr.Status.Phase = "failed"
	cr.Status.Message = issue.Error()

	err := r.client.Status().Update(r.ctx, cr)
	if err != nil {
		// Ignore conflicts, resource might just be outdated.
		if errors.IsConflict(err) {
			err = nil
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true, RequeueAfter: utils.RequeueDelay}, nil
}

func handleSucess(r *ReconcileGrafana, cr *v1alpha1.Grafana) (reconcile.Result, error) {

	cr.Status.Phase = "reconciling"
	cr.Status.Message = "success"

	err := r.client.Status().Update(r.ctx, cr)
	if err != nil {
		return handleError(r, cr, err)
	}

	log.Info("desired cluster state met")

	return reconcile.Result{RequeueAfter: utils.RequeueDelay}, nil
}

func reconcileDSProxyConfigSecret(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {
	secret := &corev1.Secret{}
	err := r.client.Get(r.ctx, client.ObjectKey{Namespace: cr.Namespace, Name: utils.DSProxyConfigSecName}, secret)
	// create/update when datasource is not common service prometheus
	//craeate
	if err != nil && errors.IsNotFound(err) {
		if secret, err = utils.DSProxyConfigSecret(cr, nil); err != nil {
			return err
		}
		if err = controllerutil.SetControllerReference(cr, secret, r.scheme); err != nil {
			return err
		}
		if err = r.client.Create(r.ctx, secret); err != nil {
			return err
		}
		log.Info("data source configuration secret is created")
		return nil
	}
	if err != nil {
		return err
	}
	//update
	if secret, err = utils.DSProxyConfigSecret(cr, secret); err != nil {
		return err
	}
	if err = controllerutil.SetControllerReference(cr, secret, r.scheme); err != nil {
		return err
	}
	if err = r.client.Update(r.ctx, secret); err != nil {
		return err
	}
	log.Info("data source configuration secret is updated")
	return nil

}

func reconcileCert(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {
	certSecretName := "ibm-monitoring-certs"
	if cr.Spec.TLSSecretName != "" {
		certSecretName = cr.Spec.TLSSecretName
	}
	cert := utils.GetCertificate(certSecretName, cr)
	if err := r.kclient.Get(r.ctx, client.ObjectKey{Name: cert.Name, Namespace: cert.Namespace}, cert); err != nil {
		if errors.IsNotFound(err) {
			//create cert
			if err := controllerutil.SetControllerReference(cr, cert, r.scheme); err != nil {
				log.Error(err, "fail to create certificate "+certSecretName)
			}
			if err := r.client.Create(r.ctx, cert); err != nil {
				log.Error(err, "fail to create certificate "+certSecretName)
				return err
			}
			log.Info("certificate " + certSecretName + " is created")
			return nil
		}
		log.Error(err, "fail to get certificate: "+certSecretName)
		return err

	}
	log.Info("certificate " + certSecretName + " exists already")

	return nil
}
func checkApplicationMonitoring(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {
	enabled, err := doCheckApplicationMonitoring(r)
	if err != nil {
		log.Error(err, "Failed to get application monitoring status")
		return err
	}
	if enabled {
		r.recorder.Eventf(cr, corev1.EventTypeNormal, "OCP application monitoring is enabled", "OCP application monitoring is enabled")

	} else {
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"OCP application monitoring is not enabled", "OCP application monitoring is not enabled. IBM application metrics can not be collected and related dashboards will not work")

	}

	return nil

}
func doCheckApplicationMonitoring(r *ReconcileGrafana) (bool, error) {
	key := client.ObjectKey{Name: "cluster-monitoring-config", Namespace: "openshift-monitoring"}
	cm := &corev1.ConfigMap{}
	err := r.kclient.Get(r.ctx, key, cm)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	configStr, exist := cm.Data["config.yaml"]
	if !exist || configStr == "" {
		return false, nil
	}
	configObj := make(map[string]interface{})
	if err = yaml.Unmarshal([]byte(configStr), &configObj); err != nil {
		return false, err
	}
	enabled, exist := configObj["enableUserWorkload"]
	if !exist {
		return false, nil

	}
	if enabled.(bool) {
		return true, nil
	}
	return false, nil

}

func cleanupCSMonitoring(r *ReconcileGrafana, cr *v1alpha1.Grafana) error {

	log.Info("Starting to cleanup old CS Monitoring resources.")

	collectdDep := utils.CollectdDeployment(cr)
	err := r.client.Delete(r.ctx, collectdDep)
	if err != nil {
		log.Info("Failed to clean old Collectd deployment or its already cleaned")
	} else {
		log.Info("old Collectd deployment is cleaned successfully")
	}

	kubestateDep := utils.KubestateDeployment(cr)
	err = r.client.Delete(r.ctx, kubestateDep)
	if err != nil {
		log.Info("Failed to clean old kube-state deployment or its already cleaned")
	} else {
		log.Info("old kube-state deployment is cleaned successfully")
	}

	nodeExporter := utils.NodeExporterDaemonSet(cr)
	err = r.client.Delete(r.ctx, nodeExporter)
	if err != nil {
		log.Info("Failed to clean old Node exporter daemonset or its already cleaned")
	} else {
		log.Info("old nodeexporter daemonset is cleaned successfully")
	}

	promOpr := utils.PrometheusOperatorDeployment(cr)
	err = r.client.Delete(r.ctx, promOpr)
	if err != nil {
		log.Info("Failed to clean old Prometheus operator deployment or its already cleaned")
	} else {
		log.Info("old prometheus operator deployment is cleaned successfully")
	}

	promStatefulset := utils.PrometheusStatefulSet(cr)
	err = r.client.Delete(r.ctx, promStatefulset)
	if err != nil {
		log.Info("Failed to clean old prometheus statefulset or its already cleaned")
	} else {
		log.Info("old prometheus statefulset is cleaned successfully")
	}

	alertmngrStatefulset := utils.AlertManagerStatefulset(cr)
	err = r.client.Delete(r.ctx, alertmngrStatefulset)
	if err != nil {
		log.Info("Failed to clean old alert-manager statefulset or its already cleaned")
	} else {
		log.Info("old alert-manager statefulset is cleaned successfully")
	}

	mcmCtlDep := utils.McmCtlDeployment(cr)
	err = r.client.Delete(r.ctx, mcmCtlDep)
	if err != nil {
		log.Info("Failed to clean old mcmCtl deployment or its already cleaned")
	} else {
		log.Info("old mcmCtl deployment is cleaned successfully")
	}
	return err
}
