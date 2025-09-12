package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	capiframework "github.com/openshift/cluster-capi-operator/e2e/framework"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("[sig-cluster-lifecycle][OCPFeatureGate:MachineAPIMigration] MachineSet Migration MAPI Authoritative Tests", Ordered, func() {
	BeforeAll(func() {
		if platform != configv1.AWSPlatformType {
			Skip(fmt.Sprintf("Skipping tests on %s, this only support on aws", platform))
		}

		if !capiframework.IsMachineAPIMigrationEnabled(ctx, cl) {
			Skip("Skipping, this feature is only supported on MachineAPIMigration enabled clusters")
		}
	})

	var _ = Describe("Create MAPI MachineSets with MAPI Authority", Ordered, func() {
		var mapiMSAuthMAPIName = "ms-authoritativeapi-mapi"
		var existingCAPIMSAuthorityMAPIName = "capi-machineset-authoritativeapi-mapi"

		var awsMachineTemplate *awsv1.AWSMachineTemplate
		var capiMachineSet *clusterv1.MachineSet
		var mapiMachineSet *machinev1beta1.MachineSet
		var err error

		Context("with spec.authoritativeAPI: MAPI and existing CAPI MachineSet with same name", func() {
			BeforeAll(func() {
				capiMachineSet, err = createCAPIMachineSet(ctx, cl, 0, existingCAPIMSAuthorityMAPIName, "")
				Expect(err).ToNot(HaveOccurred(), "CAPI MachineSet %s creation should succeed", capiMachineSet.GetName())

				Eventually(func() error {
					awsMachineTemplate, err = capiframework.GetAWSMachineTemplateByPrefix(cl, existingCAPIMSAuthorityMAPIName, capiframework.CAPINamespace)
					return err
				}, capiframework.WaitMedium, capiframework.RetryShort).Should(Succeed(), "awsMachineTemplate should exist")

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: MAPI and existing CAPI MachineSet with same name' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*machinev1beta1.MachineSet{},
					)
				})
			})

			// https://issues.redhat.com/browse/OCPCLOUD-2641
			PIt("should reject creation of MAPI MachineSet with same name as existing CAPI MachineSet", func() {
				By("Creating a same name MAPI MachineSet")
				mapiMachineSet, err = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, existingCAPIMSAuthorityMAPIName, machinev1beta1.MachineAuthorityMachineAPI, machinev1beta1.MachineAuthorityMachineAPI)
				Expect(err).To(HaveOccurred(), "denied request to create MAPI MachineSet %s", mapiMachineSet.GetName())
			})
		})

		Context("with spec.authoritativeAPI: MAPI and when no existing CAPI MachineSet with same name", func() {
			BeforeAll(func() {
				mapiMachineSet, err = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, mapiMSAuthMAPIName, machinev1beta1.MachineAuthorityMachineAPI, machinev1beta1.MachineAuthorityMachineAPI)
				Expect(err).ToNot(HaveOccurred(), "MAPI MachineSet %s creation should succeed", mapiMachineSet.GetName())

				Eventually(func() error {
					awsMachineTemplate, err = capiframework.GetAWSMachineTemplateByPrefix(cl, mapiMSAuthMAPIName, capiframework.CAPINamespace)
					return err
				}, capiframework.WaitMedium, capiframework.RetryMedium).Should(Succeed(), "awsMachineTemplate should exist")

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: MAPI and when no existing CAPI MachineSet with same name' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*machinev1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("should find MAPI MachineSet .status.authoritativeAPI to equal MAPI", func() {
				Eventually(komega.Object(mapiMachineSet)).Should(HaveField("Status.AuthoritativeAPI", Equal(machinev1beta1.MachineAuthorityMachineAPI)))
			})

			It("should verify that MAPI MachineSet paused condition is False", func() {
				verifyMachineSetMAPIPausedCondition(mapiMachineSet, machinev1beta1.MachineAuthorityMachineAPI)
			})

			It("should verify that MAPI MachineSet Synchronized condition is True", func() {
				verifyMachineSetSynchronizedCondition(mapiMachineSet, machinev1beta1.MachineAuthorityMachineAPI)
			})

			It("should find that MAPI MachineSet has a CAPI MachineSet mirror", func() {
				Eventually(func() error {
					capiMachineSet, err = capiframework.GetMachineSet(cl, mapiMSAuthMAPIName, capiframework.CAPINamespace)
					return err
				}, capiframework.WaitMedium, capiframework.RetryMedium).Should(Succeed(), "CAPI MachineSet mirror should exist")
			})

			It("should verify that the mirror CAPI MachineSet has Paused condition True", func() {
				verifyMachineSetCAPIPausedCondition(capiMachineSet, machinev1beta1.MachineAuthorityMachineAPI)
			})
		})
	})
})