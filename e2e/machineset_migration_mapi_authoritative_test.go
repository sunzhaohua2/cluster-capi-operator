package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	mapiv1beta1 "github.com/openshift/api/machine/v1beta1"
	mapiframework "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	capiframework "github.com/openshift/cluster-capi-operator/e2e/framework"
	"k8s.io/utils/ptr"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("[sig-cluster-lifecycle][OCPFeatureGate:MachineAPIMigration] MachineSet Migration MAPI Authoritative Tests", Ordered, func() {
	var k komega.Komega

	BeforeAll(func() {
		if platform != configv1.AWSPlatformType {
			Skip(fmt.Sprintf("Skipping tests on %s, this is only supported on AWS", platform))
		}

		if !capiframework.IsMachineAPIMigrationEnabled(ctx, cl) {
			Skip("Skipping, this feature is only supported on MachineAPIMigration enabled clusters")
		}

		k = komega.New(cl)
	})

	var _ = Describe("Create MAPI MachineSets", Ordered, func() {
		var mapiMSAuthMAPIName = "ms-authoritativeapi-mapi"
		var existingCAPIMSAuthorityMAPIName = "capi-machineset-authoritativeapi-mapi"

		var awsMachineTemplate *awsv1.AWSMachineTemplate
		var capiMachineSet *clusterv1.MachineSet
		var mapiMachineSet *mapiv1beta1.MachineSet

		Context("with spec.authoritativeAPI: MachineAPI and existing CAPI MachineSet with same name", func() {
			BeforeAll(func() {
				capiMachineSet = createCAPIMachineSet(ctx, cl, 0, existingCAPIMSAuthorityMAPIName, "")
				awsMachineTemplate = waitForAWSMachineTemplate(cl, existingCAPIMSAuthorityMAPIName)

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: MachineAPI and existing CAPI MachineSet with same name' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{},
					)
				})
			})

			// https://issues.redhat.com/browse/OCPCLOUD-3188
			PIt("should reject creation of MAPI MachineSet with same name as existing CAPI MachineSet", func() {
				By("Creating a same name MAPI MachineSet")
				createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, existingCAPIMSAuthorityMAPIName, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
			})
		})

		Context("with spec.authoritativeAPI: MachineAPI and when no existing CAPI MachineSet with same name", func() {
			BeforeAll(func() {
				mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, mapiMSAuthMAPIName, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
				capiMachineSet = waitForCAPIMachineSetMirror(cl, mapiMSAuthMAPIName)
				awsMachineTemplate = waitForAWSMachineTemplate(cl, mapiMSAuthMAPIName)

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: MachineAPI and when no existing CAPI MachineSet with same name' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("should find MAPI MachineSet .status.authoritativeAPI to equal MAPI", func() {
				verifyMachineSetAuthoritative(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should verify that MAPI MachineSet Paused condition is False", func() {
				verifyMachineSetPausedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should verify that MAPI MachineSet Synchronized condition is True", func() {
				verifyMAPIMachineSetSynchronizedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should find that MAPI MachineSet has a CAPI MachineSet mirror", func() {
				waitForCAPIMachineSetMirror(cl, mapiMSAuthMAPIName)
			})

			It("should verify that the mirror CAPI MachineSet has Paused condition True", func() {
				verifyMachineSetPausedCondition(capiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
			})
		})
	})

	var _ = Describe("Scale MAPI MachineSets", Ordered, func() {
		var mapiMSAuthMAPIName = "ms-authoritativeapi-mapi"
		var mapiMSAuthMAPICAPI = "ms-mapi-machine-capi"

		var awsMachineTemplate *awsv1.AWSMachineTemplate
		var capiMachineSet *clusterv1.MachineSet
		var mapiMachineSet *mapiv1beta1.MachineSet
		var firstMAPIMachine *mapiv1beta1.Machine
		var secondMAPIMachine *mapiv1beta1.Machine

		Context("with spec.authoritativeAPI: MachineAPI", Ordered, func() {
			BeforeAll(func() {
				mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 1, mapiMSAuthMAPIName, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
				capiMachineSet, awsMachineTemplate = waitForMAPIMachineSetMirrors(cl, mapiMSAuthMAPIName)

				mapiMachines, err := mapiframework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				Expect(mapiMachines).ToNot(BeEmpty(), "no MAPI Machines found")

				capiMachines := capiframework.GetMachinesFromMachineSet(cl, capiMachineSet)
				Expect(capiMachines).ToNot(BeEmpty(), "no CAPI Machines found")
				Expect(capiMachines[0].Name).To(Equal(mapiMachines[0].Name))
				firstMAPIMachine = mapiMachines[0]

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: MachineAPI' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("should be able scale MAPI MachineSet to 2 replicas successfully", func() {
				By("Scaling up MAPI MachineSet to 2 replicas")
				Expect(mapiframework.ScaleMachineSet(mapiMachineSet.GetName(), 2)).To(Succeed(), "should be able to scale up MAPI MachineSet")
				mapiframework.WaitForMachineSet(ctx, cl, mapiMSAuthMAPIName)
				verifyMachinesetReplicas(mapiMachineSet, 2)
				verifyMachinesetReplicas(capiMachineSet, 2)

				By("Verifying a new MAPI Machine is created and Paused condition is False")
				var err error
				secondMAPIMachine, err = mapiframework.GetLatestMachineFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				verifyMachineRunning(cl, secondMAPIMachine)
				verifyMachineAuthoritative(secondMAPIMachine, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMachinePausedCondition(secondMAPIMachine, mapiv1beta1.MachineAuthorityMachineAPI)

				By("Verifying there is a non-authoritative CAPI Machine mirror for the MAPI Machine and its Paused condition is True")
				capiMachine := capiframework.GetNewestMachineFromMachineSet(cl, capiMachineSet)
				verifyMachinePausedCondition(capiMachine, mapiv1beta1.MachineAuthorityMachineAPI)

				By("Verifying CAPI MachineSet status.replicas is set to 2")
				verifyMachinesetReplicas(capiMachineSet, 2)
			})

			It("should succeed switching MAPI MachineSet AuthoritativeAPI to ClusterAPI", func() {
				switchMachineSetAuthoritativeAPI(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachineSetPausedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachineSetPausedCondition(capiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMAPIMachineSetSynchronizedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should succeed scaling up CAPI MachineSet to 3, after the switch of AuthoritativeAPI to ClusterAPI", func() {
				By("Scaling up CAPI MachineSet to 3")
				capiframework.ScaleCAPIMachineSet(mapiMSAuthMAPIName, 3, capiframework.CAPINamespace)

				By("Verifying MachineSet status.replicas is set to 3")
				verifyMachinesetReplicas(capiMachineSet, 3)
				verifyMachinesetReplicas(mapiMachineSet, 3)

				By("Verifying a new CAPI Machine is running and Paused condition is False")
				capiMachine := capiframework.GetNewestMachineFromMachineSet(cl, capiMachineSet)
				verifyMachineRunning(cl, capiMachine)
				verifyMachinePausedCondition(capiMachine, mapiv1beta1.MachineAuthorityClusterAPI)

				By("Verifying there is a non-authoritative, paused MAPI Machine mirror for the new CAPI Machine")
				mapiMachine, err := mapiframework.GetLatestMachineFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				verifyMachineAuthoritative(mapiMachine, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachinePausedCondition(mapiMachine, mapiv1beta1.MachineAuthorityClusterAPI)

				By("Verifying old Machines still exist and authority on them is still MachineAPI")
				verifyMachineAuthoritative(firstMAPIMachine, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMachineAuthoritative(secondMAPIMachine, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should succeed scaling down CAPI MachineSet to 1, after the switch of AuthoritativeAPI to ClusterAPI", func() {
				By("Scaling down CAPI MachineSet to 1")
				capiframework.ScaleCAPIMachineSet(mapiMSAuthMAPIName, 1, capiframework.CAPINamespace)

				By("Verifying both CAPI MachineSet and its MAPI MachineSet mirror are scaled down to 1")
				verifyMachinesetReplicas(capiMachineSet, 1)
				verifyMachinesetReplicas(mapiMachineSet, 1)
			})

			It("should succeed in switching back the AuthoritativeAPI to MachineAPI after the initial switch to ClusterAPI", func() {
				switchMachineSetAuthoritativeAPI(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMachineSetPausedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMachineSetPausedCondition(capiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMAPIMachineSetSynchronizedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should delete both MAPI and CAPI MachineSets/Machines and InfraMachineTemplate when deleting MAPI MachineSet", func() {
				Expect(mapiframework.DeleteMachineSets(cl, mapiMachineSet)).To(Succeed(), "Should be able to delete test MachineSet")
				capiframework.WaitForMachineSetsDeleted(cl, capiMachineSet)
				mapiframework.WaitForMachineSetsDeleted(ctx, cl, mapiMachineSet)
				verifyResourceRemoved(awsMachineTemplate)
			})
		})

		Context("with spec.authoritativeAPI: MachineAPI, spec.template.spec.authoritativeAPI: ClusterAPI", Ordered, func() {
			BeforeAll(func() {
				mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, mapiMSAuthMAPICAPI, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityClusterAPI)
				capiMachineSet, awsMachineTemplate = waitForMAPIMachineSetMirrors(cl, mapiMSAuthMAPICAPI)

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: MachineAPI, spec.template.spec.authoritativeAPI: ClusterAPI' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("should create an authoritative CAPI Machine when scaling MAPI MachineSet to 1 replicas", func() {
				By("Scaling up MAPI MachineSet to 1 replicas")
				Expect(mapiframework.ScaleMachineSet(mapiMachineSet.GetName(), 1)).To(Succeed(), "should be able to scale up MAPI MachineSet")
				capiframework.WaitForMachineSet(cl, mapiMSAuthMAPICAPI, capiframework.CAPINamespace)
				verifyMachinesetReplicas(mapiMachineSet, 1)
				verifyMachinesetReplicas(capiMachineSet, 1)

				By("Verifying MAPI Machine is created and .status.authoritativeAPI to equal CAPI")
				mapiMachine, err := mapiframework.GetLatestMachineFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				verifyMachineAuthoritative(mapiMachine, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachinePausedCondition(mapiMachine, mapiv1beta1.MachineAuthorityClusterAPI)

				By("Verifying CAPI Machine is created and Paused condition is False and provisions a running Machine")
				capiMachine := capiframework.GetNewestMachineFromMachineSet(cl, capiMachineSet)
				verifyMachineRunning(cl, capiMachine)
				verifyMachinePausedCondition(capiMachine, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should delete both MAPI and CAPI MachineSets/Machines and InfraMachineTemplate when deleting MAPI MachineSet", func() {
				Expect(mapiframework.DeleteMachineSets(cl, mapiMachineSet)).To(Succeed(), "Should be able to delete test MachineSet")
				capiframework.WaitForMachineSetsDeleted(cl, capiMachineSet)
				mapiframework.WaitForMachineSetsDeleted(ctx, cl, mapiMachineSet)
				verifyResourceRemoved(awsMachineTemplate)
			})
		})
	})

	var _ = Describe("Update MachineSets", Ordered, func() {
		var mapiMSAuthMAPIName = "ms-authoritativeapi-mapi"
		var mapiMachineSet *mapiv1beta1.MachineSet
		var capiMachineSet *clusterv1.MachineSet
		var awsMachineTemplate *awsv1.AWSMachineTemplate
		var newAWSMachineTemplate *awsv1.AWSMachineTemplate

		BeforeAll(func() {
			mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, mapiMSAuthMAPIName, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
			capiMachineSet, awsMachineTemplate = waitForMAPIMachineSetMirrors(cl, mapiMSAuthMAPIName)

			DeferCleanup(func() {
				By("Cleaning up 'Update MachineSet' resources")
				cleanupMachineSetTestResources(
					ctx,
					cl,
					[]*clusterv1.MachineSet{capiMachineSet},
					[]*awsv1.AWSMachineTemplate{awsMachineTemplate, newAWSMachineTemplate},
					[]*mapiv1beta1.MachineSet{mapiMachineSet},
				)
			})
		})

		Context("when MAPI MachineSet with spec.authoritativeAPI: MachineAPI and replicas 0", Ordered, func() {
			It("should reject update when attempting scaling of the CAPI MachineSet mirror", func() {
				By("Scaling up CAPI MachineSet to 1 should be rejected")
				capiframework.ScaleCAPIMachineSet(mapiMSAuthMAPIName, 1, capiframework.CAPINamespace)
				capiMachineSet = capiframework.GetMachineSet(cl, mapiMSAuthMAPIName, capiframework.CAPINamespace)
				verifyMachinesetReplicas(capiMachineSet, 0)
			})

			It("should reject update when attempting to change the spec of the CAPI MachineSet mirror", func() {
				By("Updating CAPI mirror spec (such as Deletion.Order)")
				Eventually(k.Update(capiMachineSet, func() {
					capiMachineSet.Spec.Deletion = clusterv1.MachineSetDeletionSpec{
						Order: clusterv1.OldestMachineSetDeletionOrder,
					}
				}), capiframework.WaitMedium, capiframework.RetryShort).Should(Succeed(), "Failed to update CAPI MachineSet Deletion.Order")

				By("Verifying both MAPI and CAPI MachineSet spec value are restored to original value")
				Eventually(k.Object(mapiMachineSet), capiframework.WaitShort, capiframework.RetryShort).Should(HaveField("Spec.DeletePolicy", SatisfyAny(BeEmpty(), Equal("Random"))), "Should have DeletePolicy be either empty or 'Random'")
				Eventually(k.Object(capiMachineSet), capiframework.WaitShort, capiframework.RetryShort).Should(HaveField("Spec.Deletion.Order", Equal(clusterv1.RandomMachineSetDeletionOrder)), "Should have Deletion.Order be 'Random'")
			})

			It("should create a new InfraTemplate when update MAPI MachineSet providerSpec", func() {
				By("Updating MAPI MachineSet providerSpec InstanceType to m5.large")
				newInstanceType := "m5.large"
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.InstanceType = newInstanceType
				})

				By("Waiting for new InfraTemplate to be created")
				originalAWSMachineTemplateName := capiMachineSet.Spec.Template.Spec.InfrastructureRef.Name
				capiMachineSet = capiframework.GetMachineSet(cl, mapiMSAuthMAPIName, capiframework.CAPINamespace)
				Eventually(k.Object(capiMachineSet), capiframework.WaitMedium, capiframework.RetryMedium).Should(HaveField("Spec.Template.Spec.InfrastructureRef.Name", Not(Equal(originalAWSMachineTemplateName))), "Should have InfraTemplate name changed")

				By("Verifying new InfraTemplate has the updated InstanceType")
				var err error
				newAWSMachineTemplate, err = capiframework.GetAWSMachineTemplateByPrefix(cl, mapiMSAuthMAPIName, capiframework.CAPINamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to get new awsMachineTemplate  %s", newAWSMachineTemplate)
				Expect(newAWSMachineTemplate.Spec.Template.Spec.InstanceType).To(Equal(newInstanceType))

				By("Verifying the old InfraTemplate is deleted")
				verifyResourceRemoved(awsMachineTemplate)
			})
		})

		Context("when switching MAPI MachineSet spec.authoritativeAPI to ClusterAPI", Ordered, func() {
			BeforeAll(func() {
				switchMachineSetAuthoritativeAPI(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMAPIMachineSetSynchronizedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should be rejected when scaling MAPI MachineSet", func() {
				By("Scaling up MAPI MachineSet to 1")
				Expect(mapiframework.ScaleMachineSet(mapiMSAuthMAPIName, 1)).To(Succeed(), "should allow scaling MAPI MachineSet")

				By("Verifying MAPI MachineSet replicas is restored to original value 0")
				verifyMachinesetReplicas(mapiMachineSet, 0)
				verifyMachinesetReplicas(capiMachineSet, 0)
			})

			It("should be rejected when when updating providerSpec of MAPI MachineSet", func() {
				By("Getting the current MAPI MachineSet providerSpec InstanceType")
				originalSpec := getAWSProviderSpecFromMachineSet(mapiMachineSet)

				By("Updating the MAPI MachineSet providerSpec InstanceType")
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.InstanceType = "m5.xlarge"
				})

				By("Verifying MAPI MachineSet instanceType is restored to original value")
				verifyMAPIMachineSetProviderSpec(mapiMachineSet, HaveField("InstanceType", Equal(originalSpec.InstanceType)))
			})

			It("should update MAPI MachineSet and remove old InfraTemplate when CAPI MachineSet points to new InfraTemplate", func() {
				By("Creating a new awsMachineTemplate with different spec")
				newInstanceType := "m6.xlarge"
				originalAWSMachineTemplateName := capiMachineSet.Spec.Template.Spec.InfrastructureRef.Name
				newAWSMachineTemplate = createAWSMachineTemplate(ctx, cl, originalAWSMachineTemplateName, func(spec *awsv1.AWSMachineSpec) {
					spec.InstanceType = newInstanceType
				})

				By("Updating CAPI MachineSet to point to the new InfraTemplate")
				updateCAPIMachineSetInfraTemplate(capiMachineSet, newAWSMachineTemplate.Name)

				By("Verifying the MAPI MachineSet is updated to reflect the new template")
				var err error
				mapiMachineSet, err = mapiframework.GetMachineSet(ctx, cl, mapiMSAuthMAPIName)
				Expect(err).ToNot(HaveOccurred(), "failed to refresh MAPI MachineSet")
				Eventually(k.Object(mapiMachineSet), capiframework.WaitMedium, capiframework.RetryMedium).Should(
					HaveField("Spec.Template.Spec.ProviderSpec.Value.Raw", ContainSubstring(newInstanceType)),
					"Should have MAPI MachineSet providerSpec updated to reflect the new InfraTemplate with InstanceType %s", newInstanceType,
				)
			})
		})
	})

	var _ = Describe("Field Conversion Tests", func() {
		var mapiMSFieldTestName = "ms-field-conversion-test"
		var mapiMachineSet *mapiv1beta1.MachineSet
		var capiMachineSet *clusterv1.MachineSet
		var awsMachineTemplate *awsv1.AWSMachineTemplate

		BeforeAll(func() {
			mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, mapiMSFieldTestName, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
			capiMachineSet, awsMachineTemplate = waitForMAPIMachineSetMirrors(cl, mapiMSFieldTestName)

			DeferCleanup(func() {
				By("Cleaning up 'Field Conversion Tests' resources")
				cleanupMachineSetTestResources(
					ctx,
					cl,
					[]*clusterv1.MachineSet{capiMachineSet},
					[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
					[]*mapiv1beta1.MachineSet{mapiMachineSet},
				)
			})
		})

		Context("when MAPI MachineSet has all supported fields", func() {
			It("should convert Tags successfully", func() {
				testTags := []mapiv1beta1.TagSpecification{
					{Name: "Environment", Value: "production"},
					{Name: "Number", Value: "123"},
					{Name: "Special_Characters", Value: "test@example.com:8080/path?query=value&foo=bar"},
				}
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.Tags = testTags
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) int {
						return len(t.Spec.Template.Spec.AdditionalTags)
					},
					Equal(3),
					"Should have converted Tags to CAPI",
				)
			})

			It("should convert PlacementGroupName successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.PlacementGroupName = "prod-cluster-pg-partition-az1"
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) string {
						return t.Spec.Template.Spec.PlacementGroupName
					},
					Equal("prod-cluster-pg-partition-az1"),
					"Should have converted PlacementGroupName to CAPI",
				)
			})

			It("should convert PlacementGroupPartition successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.PlacementGroupPartition = ptr.To(int32(3))
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) int64 {
						return t.Spec.Template.Spec.PlacementGroupPartition
					},
					Equal(int64(3)),
					"Should have converted PlacementGroupPartition to CAPI",
				)
			})

			It("should convert Tenancy successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.Placement.Tenancy = mapiv1beta1.DedicatedTenancy
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) string {
						return t.Spec.Template.Spec.Tenancy
					},
					Equal(string(mapiv1beta1.DedicatedTenancy)),
					"Should have converted Tenancy to CAPI",
				)
			})

			It("should convert NetworkInterfaceType successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.NetworkInterfaceType = mapiv1beta1.AWSEFANetworkInterfaceType
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) awsv1.NetworkInterfaceType {
						return t.Spec.Template.Spec.NetworkInterfaceType
					},
					Equal(awsv1.NetworkInterfaceTypeEFAWithENAInterface),
					"Should have converted NetworkInterfaceType to CAPI",
				)
			})

			It("should convert SpotMarketOptions successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.SpotMarketOptions = &mapiv1beta1.SpotMarketOptions{
						MaxPrice: ptr.To("0.05"),
					}
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) *awsv1.SpotMarketOptions {
						return t.Spec.Template.Spec.SpotMarketOptions
					},
					Not(BeNil()),
					"Should have converted SpotMarketOptions to CAPI",
				)
			})

			It("should convert MetadataServiceOptions successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.MetadataServiceOptions = mapiv1beta1.MetadataServiceOptions{
						Authentication: mapiv1beta1.MetadataServiceAuthenticationRequired,
					}
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) *awsv1.InstanceMetadataOptions {
						return t.Spec.Template.Spec.InstanceMetadataOptions
					},
					Not(BeNil()),
					"Should have converted MetadataServiceOptions to CAPI",
				)
			})

			It("should convert SecurityGroups successfully", func() {
				testSGID := "sg-0123456789abcdef0"
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.SecurityGroups = []mapiv1beta1.AWSResourceReference{
						{ID: ptr.To(testSGID)},
					}
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) int {
						return len(t.Spec.Template.Spec.AdditionalSecurityGroups)
					},
					Equal(1),
					"Should have converted SecurityGroups to CAPI",
				)
			})

			It("should convert Subnet successfully", func() {
				testSubnetID := "subnet-0123456789abcdef0"
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.Subnet = mapiv1beta1.AWSResourceReference{
						ID: ptr.To(testSubnetID),
					}
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) string {
						if t.Spec.Template.Spec.Subnet == nil || t.Spec.Template.Spec.Subnet.ID == nil {
							return ""
						}
						return *t.Spec.Template.Spec.Subnet.ID
					},
					Equal(testSubnetID),
					"Should have converted Subnet to CAPI",
				)
			})

			It("should convert KeyName successfully", func() {
				testKeyName := "openshift-prod-us-east-1-ssh-key"
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.KeyName = ptr.To(testKeyName)
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) string {
						if t.Spec.Template.Spec.SSHKeyName == nil {
							return ""
						}
						return *t.Spec.Template.Spec.SSHKeyName
					},
					Equal(testKeyName),
					"Should have converted KeyName to CAPI",
				)
			})

			// check bug
			It("should convert BlockDevices RootVolume successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.BlockDevices = []mapiv1beta1.BlockDeviceMappingSpec{
						{
							EBS: &mapiv1beta1.EBSBlockDeviceSpec{
								VolumeSize:   ptr.To(int64(500)),
								VolumeType:   ptr.To("gp3"),
								Iops:         ptr.To(int64(16000)),
								ThroughputMib: ptr.To(int32(1000)),
								Encrypted:    ptr.To(true),
							},
						},
					}
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) int64 {
						if t.Spec.Template.Spec.RootVolume == nil {
							return 0
						}
						return t.Spec.Template.Spec.RootVolume.Size
					},
					Equal(int64(500)),
					"Should have converted BlockDevices RootVolume to CAPI",
				)
			})

			It("should convert BlockDevices NonRootVolumes successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.BlockDevices = []mapiv1beta1.BlockDeviceMappingSpec{
						{
							DeviceName: ptr.To("/dev/sdf"),
							EBS: &mapiv1beta1.EBSBlockDeviceSpec{
								VolumeSize:   ptr.To(int64(2000)),
								VolumeType:   ptr.To("io2"),
								Iops:         ptr.To(int64(64000)),
								Encrypted:    ptr.To(true),
							},
						},
					}
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) int {
						return len(t.Spec.Template.Spec.NonRootVolumes)
					},
					Equal(1),
					"Should have converted BlockDevices NonRootVolumes to CAPI",
				)
			})

			It("should convert CapacityReservationID successfully", func() {
				testCapacityReservationID := "cr-0123456789abcdef0"
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.CapacityReservationID = testCapacityReservationID
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) string {
						if t.Spec.Template.Spec.CapacityReservationID == nil {
							return ""
						}
						return *t.Spec.Template.Spec.CapacityReservationID
					},
					Equal(testCapacityReservationID),
					"Should have converted CapacityReservationID to CAPI",
				)
			})

			It("should convert MarketType successfully", func() {
				updateAWSMachineSetProviderSpec(ctx, cl, mapiMachineSet, func(providerSpec *mapiv1beta1.AWSMachineProviderConfig) {
					providerSpec.MarketType = mapiv1beta1.MarketTypeSpot
				})
				expectAWSMachineTemplateField(cl, mapiMSFieldTestName,
					func(t *awsv1.AWSMachineTemplate) awsv1.MarketType {
						return t.Spec.Template.Spec.MarketType
					},
					Equal(awsv1.MarketTypeSpot),
					"Should have converted MarketType to CAPI",
				)
			})
		})
	})
})
