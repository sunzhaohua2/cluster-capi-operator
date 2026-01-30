package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"
	mapiframework "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	capiframework "github.com/openshift/cluster-capi-operator/e2e/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("[sig-cluster-lifecycle] AuthoritativeAPI Defaulting and Status Tests on Non-AWS Platforms", Ordered, func() {
	BeforeAll(func() {
		if platform == configv1.AWSPlatformType {
			Skip("Skipping on AWS, dedicated migration tests cover this functionality")
		}

		if !capiframework.IsMachineAPIMigrationEnabled(ctx, cl) {
			Skip("Skipping, this feature is only supported on MachineAPIMigration enabled clusters")
		}
	})

	var _ = Describe("MachineSet AuthoritativeAPI Defaulting and Status", Ordered, func() {
		Context("MachineSet with spec.authoritativeAPI: MachineAPI", func() {
			var mapiMachineSetMAPI *mapiv1beta1.MachineSet
			var mapiMSAuthMAPIName = "ms-authoritativeapi-mapi"

			BeforeAll(func() {
				By("Creating a MAPI MachineSet with authoritativeAPI: MachineAPI")
				mapiMachineSetMAPI = createMAPIMachineSetWithAuthoritativeAPI(
					ctx,
					cl,
					0,
					mapiMSAuthMAPIName,
					mapiv1beta1.MachineAuthorityMachineAPI,
					mapiv1beta1.MachineAuthorityMachineAPI,
				)

				DeferCleanup(func() {
					By("Cleaning up Context 'MachineSet with spec.authoritativeAPI: MachineAPI' resources")
					cleanupMachineSetTestResources(ctx, cl, nil, nil, []*mapiv1beta1.MachineSet{mapiMachineSetMAPI})
				})
			})

			It("should default .status.authoritativeAPI to .spec.authoritativeAPI (MachineAPI)", func() {
				By("Verifying status.authoritativeAPI is set to MachineAPI")
				verifyMachineSetAuthoritative(mapiMachineSetMAPI, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should have MAPI MachineSet Paused condition False", func() {
				By("Verifying MAPI MachineSet is not paused")
				verifyMachineSetPausedCondition(mapiMachineSetMAPI, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should not create any CAPI resources in openshift-cluster-api namespace", func() {
				By("Verifying no CAPI MachineSet exists")
				capiMachineSet := &clusterv1.MachineSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mapiMachineSetMAPI.Name,
						Namespace: capiframework.CAPINamespace,
					},
				}
				Consistently(komega.Get(capiMachineSet)).Should(Not(Succeed()), "Should not find CAPI MachineSet on non-migrated platform")
			})
		})

		Context("MachineSet with spec.authoritativeAPI: ClusterAPI", func() {
			var mapiMachineSetCAPI *mapiv1beta1.MachineSet
			var mapiMSAuthCAPIName = "ms-authoritativeapi-capi"

			BeforeAll(func() {
				By("Creating a MAPI MachineSet with authoritativeAPI: ClusterAPI")
				mapiMachineSetCAPI = createMAPIMachineSetWithAuthoritativeAPI(
					ctx,
					cl,
					1,
					mapiMSAuthCAPIName,
					mapiv1beta1.MachineAuthorityClusterAPI,
					mapiv1beta1.MachineAuthorityClusterAPI,
				)

				DeferCleanup(func() {
					By("Cleaning up Context 'MachineSet with spec.authoritativeAPI: ClusterAPI' resources")
					cleanupMachineSetTestResources(ctx, cl, nil, nil, []*mapiv1beta1.MachineSet{mapiMachineSetCAPI})
				})
			})

			It("should default .status.authoritativeAPI to .spec.authoritativeAPI (ClusterAPI)", func() {
				By("Verifying status.authoritativeAPI is set to ClusterAPI")
				verifyMachineSetAuthoritative(mapiMachineSetCAPI, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should have MAPI MachineSet Paused condition True", func() {
				By("Verifying MAPI MachineSet is paused")
				verifyMachineSetPausedCondition(mapiMachineSetCAPI, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should not create MAPI machines", func() {
				By("Verifying no MAPI machines are created")
				mapiMachines, err := mapiframework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSetCAPI)
				Expect(err).ToNot(HaveOccurred(), "Should have successfully listed MAPI Machines from MachineSet")
				Expect(mapiMachines).To(BeEmpty(), "Should have no MAPI machines when authoritativeAPI is ClusterAPI")
			})

			It("should not create any CAPI resources in openshift-cluster-api namespace", func() {
				By("Verifying no CAPI resources are created on non-migrated platform")
				capiMachineSet := &clusterv1.MachineSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mapiMachineSetCAPI.Name,
						Namespace: capiframework.CAPINamespace,
					},
				}
				Consistently(komega.Get(capiMachineSet)).Should(Not(Succeed()), "Should not create CAPI MachineSet on non-migrated platform even with ClusterAPI authoritative")
			})
		})
	})

	var _ = Describe("Standalone Machine AuthoritativeAPI Defaulting and Status", Ordered, func() {
		Context("Standalone Machine with spec.authoritativeAPI: MachineAPI", func() {
			var mapiMachineMAPI *mapiv1beta1.Machine
			var mapiMachineAuthMAPIName = "machine-authoritativeapi-mapi"

			BeforeAll(func() {
				By("Creating a standalone MAPI Machine with authoritativeAPI: MachineAPI")
				mapiMachineMAPI = createMAPIMachineWithAuthority(
					ctx,
					cl,
					mapiMachineAuthMAPIName,
					mapiv1beta1.MachineAuthorityMachineAPI,
				)

				DeferCleanup(func() {
					By("Cleaning up Context 'Standalone Machine with spec.authoritativeAPI: MachineAPI' resources")
					cleanupMachineResources(ctx, cl, nil, []*mapiv1beta1.Machine{mapiMachineMAPI})
				})
			})

			It("should default .status.authoritativeAPI to .spec.authoritativeAPI (MachineAPI)", func() {
				By("Verifying status.authoritativeAPI is set to MachineAPI")
				verifyMachineAuthoritative(mapiMachineMAPI, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should have MAPI Machine Paused condition False", func() {
				By("Verifying MAPI Machine is not paused")
				verifyMachinePausedCondition(mapiMachineMAPI, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should not create any CAPI resources in openshift-cluster-api namespace", func() {
				By("Verifying no CAPI Machine exists")
				capiMachine := &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mapiMachineMAPI.Name,
						Namespace: capiframework.CAPINamespace,
					},
				}
				Consistently(komega.Get(capiMachine)).Should(Not(Succeed()), "Should not find CAPI Machine on non-migrated platform")
			})
		})

		Context("Standalone Machine with spec.authoritativeAPI: ClusterAPI", func() {
			var mapiMachineCAPI *mapiv1beta1.Machine
			var mapiMachineAuthCAPIName = "machine-authoritativeapi-capi"

			BeforeAll(func() {
				By("Creating a standalone MAPI Machine with authoritativeAPI: ClusterAPI")
				mapiMachineCAPI = createMAPIMachineWithAuthority(
					ctx,
					cl,
					mapiMachineAuthCAPIName,
					mapiv1beta1.MachineAuthorityClusterAPI,
				)

				DeferCleanup(func() {
					By("Cleaning up Context 'Standalone Machine with spec.authoritativeAPI: ClusterAPI' resources")
					cleanupMachineResources(ctx, cl, nil, []*mapiv1beta1.Machine{mapiMachineCAPI})
				})
			})

			It("should default .status.authoritativeAPI to .spec.authoritativeAPI (ClusterAPI)", func() {
				By("Verifying status.authoritativeAPI is set to ClusterAPI")
				verifyMachineAuthoritative(mapiMachineCAPI, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should have MAPI Machine Paused condition True", func() {
				By("Verifying MAPI Machine is paused")
				verifyMachinePausedCondition(mapiMachineCAPI, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should not create any CAPI resources in openshift-cluster-api namespace", func() {
				By("Verifying no CAPI resources are created on non-migrated platform")
				capiMachine := &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      mapiMachineCAPI.Name,
						Namespace: capiframework.CAPINamespace,
					},
				}
				Consistently(komega.Get(capiMachine)).Should(Not(Succeed()), "Should not create CAPI Machine on non-migrated platform even with ClusterAPI authoritative")
			})
		})
	})
})
