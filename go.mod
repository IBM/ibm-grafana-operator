module github.com/IBM/ibm-monitoring-grafana-operator

go 1.15

require (
	github.com/jetstack/cert-manager v0.13.0
	github.com/operator-framework/operator-sdk v0.18.0
	sigs.k8s.io/controller-runtime v0.9.0
)

// Pinned to kubernetes-0.21.2
replace (
	k8s.io/api => k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.2
	k8s.io/apiserver => k8s.io/apiserver v0.21.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.2
	k8s.io/client-go => k8s.io/client-go v0.21.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.2
	k8s.io/code-generator => k8s.io/code-generator v0.21.2
	k8s.io/component-base => k8s.io/component-base v0.21.2
	k8s.io/cri-api => k8s.io/cri-api v0.21.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.2
	k8s.io/kubectl => k8s.io/kubectl v0.21.2
	k8s.io/kubelet => k8s.io/kubelet v0.21.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.2
	k8s.io/metrics => k8s.io/metrics v0.21.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.2
)

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

replace github.com/docker/docker => github.com/moby/moby v17.10.0-ce+incompatible // Required by Helm

// pinned to cert manager v0.10
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/jetstack/cert-manager => github.com/jetstack/cert-manager v0.10.0
)

require (
	github.com/spf13/pflag v1.0.5
	github.ibm.com/IBMPrivateCloud/grafana-dashboard-crd v1.2.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20210527164424-3c818078ee3d
	sigs.k8s.io/yaml v1.2.0
)
