package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	dynamicclient "k8s.io/client-go/dynamic"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"

	opv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/csi/csicontrollerset"
	goc "github.com/openshift/library-go/pkg/operator/genericoperatorclient"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

const (
	// Operand and operator run in the same namespace
	defaultNamespace = "openshift-cluster-csi-drivers"
	operatorName     = "kubevirt-csi-driver-operator"
	operandName      = "kubevirt-csi-driver"
	instanceName     = "csi.kubevirt.io"
)

func RunOperator(ctx context.Context, controllerConfig *controllercmd.ControllerContext) error {
	// Create clientsets and informers
	kubeClient := kubeclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, operatorName))
	dynamicClient := dynamicclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, operatorName))
	kubeInformersForNamespaces := v1helpers.NewKubeInformersForNamespaces(kubeClient, defaultNamespace, "")

	// Create driver config
	configMap, err := createDriverConfig(ctx, kubeClient)
	if err != nil {
		panic(err)
	}

	// Create YAML for driver config
	bytes, err := yaml.Marshal(configMap)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("assets/configmap.yaml", bytes, 0777)
	if err != nil {
		panic(err)
	}

	// Create GenericOperatorclient. This is used by the library-go controllers created down below
	gvr := opv1.SchemeGroupVersion.WithResource("clustercsidrivers")
	operatorClient, dynamicInformers, err := goc.NewClusterScopedOperatorClientWithConfigName(controllerConfig.KubeConfig, gvr, instanceName)
	if err != nil {
		return err
	}

	csiControllerSet := csicontrollerset.NewCSIControllerSet(
		operatorClient,
		controllerConfig.EventRecorder,
	).WithLogLevelController().WithManagementStateController(
		operandName,
		false,
	).WithStaticResourcesController(
		"KubevirtDriverStaticResources",
		kubeClient,
		kubeInformersForNamespaces,
		asset,
		[]string{
			"configmap.yaml",
			"csi-driver.yaml",
			"node-sa.yaml",
			"node-cr.yaml",
			"node-binding.yaml",
			"controller-sa.yaml",
			"controller-cr.yaml",
			"controller-binding.yaml",
			"leader-election-cr.yaml",
			"controller-leader-binding.yaml",
			"node-leader-binding.yaml",
		},
	).
		WithCredentialsRequestController(
			"KubevirtDriverCredentialsRequestController",
			defaultNamespace,
			assetPanic,
			"credentials-request.yaml",
			dynamicClient,
		).
		WithCSIDriverController(
			"KubevirtDriverController",
			instanceName,
			operandName,
			defaultNamespace,
			assetPanic,
			kubeClient,
			kubeInformersForNamespaces.InformersFor(defaultNamespace),
			csicontrollerset.WithControllerService("controller.yaml"),
			csicontrollerset.WithNodeService("node.yaml"),
		)

	if err != nil {
		return err
	}

	klog.Info("Starting the informers")
	go kubeInformersForNamespaces.Start(ctx.Done())
	go dynamicInformers.Start(ctx.Done())

	klog.Info("Starting controllerset")
	go csiControllerSet.Run(ctx, 1)

	<-ctx.Done()

	return fmt.Errorf("stopped")
}

func asset(name string) ([]byte, error) {
	return ioutil.ReadFile("assets/" + name) // Folder assets must be placed in the process's working directory
}

func assetPanic(name string) []byte {
	bytes, err := asset(name)
	if err != nil {
		panic("Fetching asset " + name + " failed. Error: " + err.Error())
	}

	return bytes
}

func createDriverConfig(ctx context.Context, kubeClient *kubeclient.Clientset) (*corev1.ConfigMap, error) {
	configMap, err := kubeClient.CoreV1().ConfigMaps("openshift-config").Get(ctx, "cloud-provider-config", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	jsonConfig, ok := configMap.Data["config"]
	if !ok {
		return nil, fmt.Errorf("Field config in ConfigMap openshift-config/cloud-provider-config is missing")
	}

	var config map[string]string
	err = json.Unmarshal([]byte(jsonConfig), &config)
	if err != nil {
		return nil, err
	}

	namespace, ok := config["namespace"]
	if !ok {
		return nil, fmt.Errorf("Missing namespace in JSON string. Check field config in ConfigMap openshift-config/cloud-provider-config")
	}

	infraID, ok := config["infraID"]
	if !ok {
		return nil, fmt.Errorf("Missing infraID in JSON string. Check field config in ConfigMap openshift-config/cloud-provider-config")
	}

	driverConfig := &corev1.ConfigMap{}

	driverConfig.APIVersion = "v1"
	driverConfig.Kind = "ConfigMap"
	driverConfig.Name = "driver-config"
	driverConfig.Namespace = defaultNamespace
	driverConfig.Data = map[string]string{
		"infraClusterNamespace":  namespace,
		"infraClusterLabelName":  fmt.Sprintf("tenantcluster-%s-machine.openshift.io", infraID),
		"infraClusterLabelValue": "owned",
	}

	return driverConfig, nil
}
