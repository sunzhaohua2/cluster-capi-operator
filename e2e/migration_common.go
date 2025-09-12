package e2e

import (
	"time"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// Shared condition types for both machine and machineset migration tests
	SynchronizedCondition machinev1beta1.ConditionType = "Synchronized"
	MAPIPausedCondition   machinev1beta1.ConditionType = "Paused"
	CAPIPausedCondition                                = clusterv1.PausedV1Beta2Condition

	// Machine test constants
	RoleLabel       = "machine.openshift.io/cluster-api-machine-role"
	DefaultTimeout  = 400 * time.Second
	DefaultInterval = 10 * time.Second
)
