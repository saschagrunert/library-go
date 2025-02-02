package csidrivercontrollerservicecontroller

import (
	appsinformersv1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"

	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	"github.com/openshift/library-go/pkg/controller/factory"
	dc "github.com/openshift/library-go/pkg/operator/deploymentcontroller"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

// CSIDriverControllerServiceController is a controller that deploys a CSI Controller Service to a given namespace.
//
// The CSI Controller Service is represented by a Deployment. The reason it's a Deployment is because this object
// can be evicted and it's shut down on node drain, which is important for master nodes. This Deployment deploys a
// pod with the CSI driver and sidecars containers (provisioner, attacher, resizer, snapshotter, liveness-probe).
//
// On every sync, this controller reads the Deployment from a static file and overrides a few fields:
//
// 1. Container image locations
//
// The controller will replace the images specified in the static file if their name follows a certain nomenclature AND its
// respective environemnt variable is set. This is a list of environment variables that the controller understands:
//
// DRIVER_IMAGE
// PROVISIONER_IMAGE
// ATTACHER_IMAGE
// RESIZER_IMAGE
// SNAPSHOTTER_IMAGE
// LIVENESS_PROBE_IMAGE
//
// The names above should be wrapped by a ${}, e.g., ${DIVER_IMAGE} in static file.
//
// 2. Log level
//
// The controller can also override the log level passed in to the CSI driver container.
// In order to do that, the placeholder ${LOG_LEVEL} from the manifest file is replaced with the value specified
// in the OperatorClient resource (Spec.LogLevel).
//
// 3. Cluster ID
//
// The placeholder ${CLUSTER_ID} specified in the static file is replaced with the cluster ID (sometimes referred as infra ID).
// This is mostly used by CSI drivers to tag volumes and snapshots so that those resources can be cleaned up on cluster deletion.
//
// This controller supports removable operands, as configured in pkg/operator/management.
//
// This controller produces the following conditions:
//
// <name>Available: indicates that the CSI Controller Service was successfully deployed and at least one Deployment replica is available.
// <name>Progressing: indicates that the CSI Controller Service is being deployed.
// <name>Degraded: produced when the sync() method returns an error.

func NewCSIDriverControllerServiceController(
	name string,
	manifest []byte,
	recorder events.Recorder,
	operatorClient v1helpers.OperatorClientWithFinalizers,
	kubeClient kubernetes.Interface,
	deployInformer appsinformersv1.DeploymentInformer,
	configInformer configinformers.SharedInformerFactory,
	optionalInformers []factory.Informer,
	optionalDeploymentHooks ...dc.DeploymentHookFunc,
) factory.Controller {
	optionalInformers = append(optionalInformers, configInformer.Config().V1().Infrastructures().Informer())
	var optionalManifestHooks []dc.ManifestHookFunc
	optionalManifestHooks = append(optionalManifestHooks, WithPlaceholdersHook(configInformer))
	var deploymentHooks []dc.DeploymentHookFunc
	deploymentHooks = append(deploymentHooks, optionalDeploymentHooks...)
	deploymentHooks = append(deploymentHooks, WithControlPlaneTopologyHook(configInformer))
	return dc.NewDeploymentController(name, manifest, recorder, operatorClient, kubeClient, deployInformer, optionalInformers, optionalManifestHooks, deploymentHooks...)
}
