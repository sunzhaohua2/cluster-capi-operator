/*
Copyright 2024 Red Hat, Inc.

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

package mapi2capi

import (
	"fmt"

	mapiv1 "github.com/openshift/api/machine/v1beta1"
	conversionutil "github.com/openshift/cluster-capi-operator/pkg/conversion/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	capiNamespace            = "openshift-cluster-api"
	workerUserDataSecretName = "worker-user-data"
	awsMachineKind           = "AWSMachine"
	awsMachineTemplateKind   = "AWSMachineTemplate"
)

var (
	// awsMachineAPIVersion is the API version for the AWSMachine API.
	// Source it from the API group version so that it is always up to date.
	awsMachineAPIVersion = capav1.GroupVersion.String() //nolint:gochecknoglobals
)

// fromMAPIMachineToCAPIMachine translates a MAPI Machine to its Core CAPI Machine correspondent.
func fromMAPIMachineToCAPIMachine(mapiMachine *mapiv1.Machine) (*capiv1.Machine, field.ErrorList) {
	var errs field.ErrorList

	capiMachine := &capiv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:        mapiMachine.Name,
			Namespace:   capiNamespace,
			Labels:      mapiMachine.Labels,
			Annotations: mapiMachine.Annotations,
			// OwnerReferences: TODO(OCPCLOUD-2716): These need to be converted so that any MachineSet owning a Machine is represented with the correct owner reference between the two APIs.
		},
		Spec: capiv1.MachineSpec{
			InfrastructureRef: corev1.ObjectReference{
				APIVersion: awsMachineAPIVersion,
				Kind:       awsMachineKind,
				Name:       mapiMachine.Name,
				Namespace:  capiNamespace,
			},
			ProviderID: mapiMachine.Spec.ProviderID,
			// Version: TODO(OCPCLOUD-2714): To be prevented by VAP.
			// FailureDomain: populated by higher level functions.
			// ClusterName: populated by higher level functions.

			// TODO(OCPCLOUD-2715): These are not present on the MAPI API, we should implement them for feature parity.
			// NodeDrainTimeout: ,
			// NodeVolumeDetachTimeout: ,
			// NodeDeletionTimeout: ,
		},
	}

	// lifecycleHooks are handled via an annotation in Cluster API.
	lifecycleAnnotations := getCAPILifecycleHookAnnotations(mapiMachine.Spec.LifecycleHooks)
	if capiMachine.Annotations == nil {
		capiMachine.Annotations = lifecycleAnnotations
	} else {
		for key, value := range lifecycleAnnotations {
			capiMachine.Annotations[key] = value
		}
	}

	if capiMachine.Labels == nil {
		capiMachine.Labels = map[string]string{}
	}

	errs = append(errs, setMAPINodeLabelsToCAPIManagedNodeLabels(field.NewPath("spec", "metadata", "labels"), mapiMachine.Spec.ObjectMeta.Labels, capiMachine.Labels)...)

	// Unused fields - Below this line are fields not used from the MAPI Machine.

	if len(mapiMachine.OwnerReferences) > 0 {
		// TODO(OCPCLOUD-2716): We should support converting CAPI MachineSet ORs to MAPI MachineSet ORs. NB working out the UID will be hard.
		errs = append(errs, field.Invalid(field.NewPath("metadata", "ownerReferences"), mapiMachine.OwnerReferences, "ownerReferences are not supported"))
	}

	// mapiMachine.Spec.AuthoritativeAPI - Ignore as this is part of the conversion mechanism.

	// metadata.labels - needs special handling
	// metadata.annotations - needs special handling

	errs = append(errs, handleUnsupportedMachineFields(mapiMachine.Spec)...)

	return capiMachine, errs
}

func setMAPINodeLabelsToCAPIManagedNodeLabels(fldPath *field.Path, mapiNodeLabels map[string]string, capiNodeLabels map[string]string) field.ErrorList {
	if len(mapiNodeLabels) == 0 {
		return field.ErrorList{}
	}

	if capiNodeLabels == nil {
		capiNodeLabels = map[string]string{}
	}

	errs := field.ErrorList{}

	// TODO(OCPCLOUD-2680): Not all the labels on the CAPI Machine are propagated down to the corresponding CAPI Node, only the "CAPI Managed ones" are.
	// These are those prefix by "node-role.kubernetes.io" or in the domains of "node-restriction.kubernetes.io" and "node.cluster.x-k8s.io".
	// See: https://github.com/kubernetes-sigs/cluster-api/pull/7173
	// and: https://github.com/fabriziopandini/cluster-api/blob/main/docs/proposals/20220927-label-sync-between-machine-and-nodes.md
	for k, v := range mapiNodeLabels {
		if !conversionutil.IsCAPIManagedLabel(k) {
			errs = append(errs, field.Invalid(fldPath.Key(k), v, "label propagation is not currently supported for this label"))
		}

		capiNodeLabels[k] = v
	}

	return errs
}

// getCAPILifecycleHookAnnotations returns the annotations that should be added to a CAPI Machine to represent the lifecycle hooks.
func getCAPILifecycleHookAnnotations(hooks mapiv1.LifecycleHooks) map[string]string {
	annotations := make(map[string]string)

	for _, hook := range hooks.PreDrain {
		annotations[fmt.Sprintf("%s/%s", capiv1.PreDrainDeleteHookAnnotationPrefix, hook.Name)] = hook.Owner
	}

	for _, hook := range hooks.PreTerminate {
		annotations[fmt.Sprintf("%s/%s", capiv1.PreTerminateDeleteHookAnnotationPrefix, hook.Name)] = hook.Owner
	}

	return annotations
}

// handleUnsupportedMachineFields checks for fields that are not supported by CAPI and returns a list of errors.
func handleUnsupportedMachineFields(spec mapiv1.MachineSpec) field.ErrorList {
	var errs field.ErrorList

	fldPath := field.NewPath("spec")

	errs = append(errs, handleUnsupportedMAPIObjectMetaFields(fldPath.Child("metadata"), spec.ObjectMeta)...)

	// TODO(OCPCLOUD-2680): Taints are not supported by CAPI. add support for them via CAPI BootstrapConfig + minimal bootstrap controller.
	if len(spec.Taints) > 0 {
		errs = append(errs, field.Invalid(fldPath.Child("taints"), spec.Taints, "taints are not currently supported"))
	}

	return errs
}

// handleUnsupportedMAPIObjectMetaFields checks for unsupported MAPI metadta fields and returns a list of errors
// if any of them are currently set.
// This is used to prevent usage of these fields in both the Machine and MachineSet specs.
func handleUnsupportedMAPIObjectMetaFields(fldPath *field.Path, objectMeta mapiv1.ObjectMeta) field.ErrorList {
	var errs field.ErrorList

	// ObjectMeta related fields should never get converted (aside from labels and annotations).
	// They are meaningless in MAPI and don't contribute to the logic of the product.
	if objectMeta.Name != "" {
		errs = append(errs, field.Invalid(fldPath.Child("name"), objectMeta.Name, "name is not supported"))
	}

	if objectMeta.GenerateName != "" {
		errs = append(errs, field.Invalid(fldPath.Child("generateName"), objectMeta.GenerateName, "generateName is not supported"))
	}

	if objectMeta.Namespace != "" {
		errs = append(errs, field.Invalid(fldPath.Child("namespace"), objectMeta.Namespace, "namespace is not supported"))
	}

	if len(objectMeta.OwnerReferences) > 0 {
		errs = append(errs, field.Invalid(fldPath.Child("ownerReferences"), objectMeta.OwnerReferences, "ownerReferences are not supported"))
	}

	return errs
}
