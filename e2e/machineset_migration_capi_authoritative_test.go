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
)

var _ = Describe("[sig-cluster-lifecycle][OCPFeatureGate:MachineAPIMigration] MachineSet Migration CAPI Authoritative Tests", Ordered, func() {
	BeforeAll(func() {
		if platform != configv1.AWSPlatformType {
			Skip(fmt.Sprintf("Skipping tests on %s, this is only supported on AWS", platform))
		}

		if !capiframework.IsMachineAPIMigrationEnabled(ctx, cl) {
			Skip("Skipping, this feature is only supported on MachineAPIMigration enabled clusters")
		}
	})

	var _ = Describe("Create MAPI MachineSets", Ordered, func() {
		var mapiMSAuthCAPIName = "ms-authoritativeapi-capi"
		var existingCAPIMSAuthorityCAPIName = "capi-machineset-authoritativeapi-capi"

		var awsMachineTemplate *awsv1.AWSMachineTemplate
		var capiMachineSet *clusterv1.MachineSet
		var mapiMachineSet *mapiv1beta1.MachineSet
		var instanceType = "m5.large"

		Context("with spec.authoritativeAPI: ClusterAPI and existing CAPI MachineSet with same name", func() {
			BeforeAll(func() {
				capiMachineSet = createCAPIMachineSet(ctx, cl, 0, existingCAPIMSAuthorityCAPIName, instanceType)

				By("Creating a same name MAPI MachineSet")
				mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, existingCAPIMSAuthorityCAPIName, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)
				awsMachineTemplate = waitForAWSMachineTemplate(cl, existingCAPIMSAuthorityCAPIName)

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: ClusterAPI and existing CAPI MachineSet with same name' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("should verify that MAPI MachineSet has Paused condition True", func() {
				verifyMachineSetPausedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			// bug https://issues.redhat.com/browse/OCPBUGS-55337
			PIt("should verify that the non-authoritative MAPI MachineSet providerSpec has been updated to reflect the authoritative CAPI MachineSet mirror values", func() {
				verifyMAPIMachineSetProviderSpec(mapiMachineSet, HaveField("InstanceType", Equal(instanceType)))
			})
		})

		Context("with spec.authoritativeAPI: ClusterAPI and no existing CAPI MachineSet with same name", func() {
			BeforeAll(func() {
				mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, mapiMSAuthCAPIName, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)
				capiMachineSet = waitForCAPIMachineSetMirror(cl, mapiMSAuthCAPIName)
				awsMachineTemplate = waitForAWSMachineTemplate(cl, mapiMSAuthCAPIName)

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: ClusterAPI and no existing CAPI MachineSet with same name' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("should find MAPI MachineSet .status.authoritativeAPI to equal CAPI", func() {
				verifyMachineSetAuthoritative(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should verify that MAPI MachineSet Paused condition is True", func() {
				verifyMachineSetPausedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should verify that MAPI MachineSet Synchronized condition is True", func() {
				verifyMAPIMachineSetSynchronizedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should verify that the non-authoritative MAPI MachineSet has an authoritative CAPI MachineSet mirror", func() {
				waitForCAPIMachineSetMirror(cl, mapiMSAuthCAPIName)
			})

			It("should verify that CAPI MachineSet has Paused condition False", func() {
				verifyMachineSetPausedCondition(capiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})
		})
	})

	var _ = Describe("Scale MAPI MachineSets", Ordered, func() {
		var mapiMSAuthCAPIName = "ms-authoritativeapi-capi"

		var awsMachineTemplate *awsv1.AWSMachineTemplate
		var capiMachineSet *clusterv1.MachineSet
		var mapiMachineSet *mapiv1beta1.MachineSet
		var firstMAPIMachine *mapiv1beta1.Machine
		var secondMAPIMachine *mapiv1beta1.Machine

		Context("with spec.authoritativeAPI: ClusterAPI", Ordered, func() {
			BeforeAll(func() {
				mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 1, mapiMSAuthCAPIName, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)
				capiMachineSet, awsMachineTemplate = waitForMAPIMachineSetMirrors(cl, mapiMSAuthCAPIName)

				mapiMachines, err := mapiframework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				Expect(mapiMachines).ToNot(BeEmpty(), "no MAPI Machines found")

				capiMachines := capiframework.GetMachinesFromMachineSet(cl, capiMachineSet)
				Expect(capiMachines).ToNot(BeEmpty(), "no CAPI Machines found")
				Expect(capiMachines[0].Name).To(Equal(mapiMachines[0].Name))
				firstMAPIMachine = mapiMachines[0]

				DeferCleanup(func() {
					By("Cleaning up Context 'with spec.authoritativeAPI: ClusterAPI' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("should succeed scaling CAPI MachineSet to 2 replicas", func() {
				By("Scaling up CAPI MachineSet to 2 replicas")
				capiframework.ScaleCAPIMachineSet(mapiMSAuthCAPIName, 2, capiframework.CAPINamespace)

				By("Verifying MachineSet status.replicas is set to 2")
				verifyMachinesetReplicas(capiMachineSet, 2)
				verifyMachinesetReplicas(mapiMachineSet, 2)

				By("Verifying a new CAPI Machine is created and Paused condition is False")
				capiMachineSet = capiframework.GetMachineSet(cl, mapiMSAuthCAPIName, capiframework.CAPINamespace)
				capiMachine := capiframework.GetNewestMachineFromMachineSet(cl, capiMachineSet)
				verifyMachineRunning(cl, capiMachine)
				verifyMachinePausedCondition(capiMachine, mapiv1beta1.MachineAuthorityClusterAPI)

				By("Verifying there is a non-authoritative, paused MAPI Machine mirror for the new CAPI Machine")
				var err error
				secondMAPIMachine, err = mapiframework.GetLatestMachineFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				verifyMachineAuthoritative(secondMAPIMachine, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachinePausedCondition(secondMAPIMachine, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should succeed switching MachineSet's AuthoritativeAPI to MachineAPI", func() {
				switchMachineSetAuthoritativeAPI(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMachineSetPausedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMachineSetPausedCondition(capiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMAPIMachineSetSynchronizedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityMachineAPI)
			})

			It("should succeed scaling up MAPI MachineSet to 3, after switching AuthoritativeAPI to MachineAPI", func() {
				By("Scaling up MAPI MachineSet to 3 replicas")
				Expect(mapiframework.ScaleMachineSet(mapiMSAuthCAPIName, 3)).To(Succeed(), "should be able to scale up MAPI MachineSet")

				By("Verifying MachineSet status.replicas is set to 3")
				verifyMachinesetReplicas(mapiMachineSet, 3)
				verifyMachinesetReplicas(capiMachineSet, 3)

				By("Verifying the newly requested MAPI Machine has been created and its status.authoritativeAPI is MachineAPI and its Paused condition is False")
				mapiMachine, err := mapiframework.GetLatestMachineFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				verifyMachineRunning(cl, mapiMachine)
				verifyMachineAuthoritative(mapiMachine, mapiv1beta1.MachineAuthorityMachineAPI)
				verifyMachinePausedCondition(mapiMachine, mapiv1beta1.MachineAuthorityMachineAPI)

				By("Verifying there is a non-authoritative, paused CAPI Machine mirror for the new MAPI Machine")
				capiMachine := capiframework.GetNewestMachineFromMachineSet(cl, capiMachineSet)
				verifyMachinePausedCondition(capiMachine, mapiv1beta1.MachineAuthorityMachineAPI)

				By("Verifying old Machines still exist and authority on them is still ClusterAPI")
				verifyMachineAuthoritative(firstMAPIMachine, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachineAuthoritative(secondMAPIMachine, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should succeed scaling down MAPI MachineSet to 1, after the switch of AuthoritativeAPI to MachineAPI", func() {
				By("Scaling down MAPI MachineSet to 1 replicas")
				Expect(mapiframework.ScaleMachineSet(mapiMSAuthCAPIName, 1)).To(Succeed(), "should be able to scale down MAPI MachineSet")
				verifyMachinesetReplicas(mapiMachineSet, 1)
				verifyMachinesetReplicas(capiMachineSet, 1)
			})

			It("should succeed switching back MachineSet's AuthoritativeAPI to ClusterAPI, after the initial switch to AuthoritativeAPI: MachineAPI", func() {
				switchMachineSetAuthoritativeAPI(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachineSetPausedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachineSetPausedCondition(capiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMAPIMachineSetSynchronizedCondition(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI)
			})

			It("should delete both MAPI and CAPI MachineSets/Machines and InfraMachineTemplate when deleting CAPI MachineSet", func() {
				capiframework.DeleteMachineSets(ctx, cl, capiMachineSet)
				mapiframework.WaitForMachineSetsDeleted(ctx, cl, mapiMachineSet)
				capiframework.WaitForMachineSetsDeleted(cl, capiMachineSet)
				verifyResourceRemoved(awsMachineTemplate)
			})
		})
	})

	var _ = Describe("Delete MachineSets", Ordered, func() {
		var mapiMSAuthMAPIName = "ms-authoritativeapi-mapi"
		var mapiMachineSet *mapiv1beta1.MachineSet
		var capiMachineSet *clusterv1.MachineSet
		var awsMachineTemplate *awsv1.AWSMachineTemplate

		Context("when removing non-authoritative MAPI MachineSet", Ordered, func() {
			BeforeAll(func() {
				mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 1, mapiMSAuthMAPIName, mapiv1beta1.MachineAuthorityMachineAPI, mapiv1beta1.MachineAuthorityMachineAPI)
				capiMachineSet, awsMachineTemplate = waitForMAPIMachineSetMirrors(cl, mapiMSAuthMAPIName)

				mapiMachines, err := mapiframework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(mapiMachines).ToNot(BeEmpty(), "no MAPI Machines found")
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")

				capiMachines := capiframework.GetMachinesFromMachineSet(cl, capiMachineSet)
				Expect(capiMachines).ToNot(BeEmpty(), "no CAPI Machines found")
				Expect(capiMachines[0].Name).To(Equal(mapiMachines[0].Name))

				DeferCleanup(func() {
					By("Cleaning up Context 'when removing non-authoritative MAPI MachineSet' resources")
					cleanupMachineSetTestResources(
						ctx,
						cl,
						[]*clusterv1.MachineSet{capiMachineSet},
						[]*awsv1.AWSMachineTemplate{awsMachineTemplate},
						[]*mapiv1beta1.MachineSet{mapiMachineSet},
					)
				})
			})

			It("shouldn't delete its authoritative CAPI MachineSet", func() {
				By("Switching AuthoritativeAPI to ClusterAPI")
				switchMachineSetAuthoritativeAPI(mapiMachineSet, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)

				By("Scaling up CAPI MachineSet to 2 replicas")
				capiframework.ScaleCAPIMachineSet(mapiMachineSet.GetName(), 2, capiframework.CAPINamespace)

				By("Verifying MachineSet status.replicas is set to 2")
				verifyMachinesetReplicas(capiMachineSet, 2)
				verifyMachinesetReplicas(mapiMachineSet, 2)

				By("Verifying new CAPI Machine is running")
				capiMachine := capiframework.GetNewestMachineFromMachineSet(cl, capiMachineSet)
				verifyMachineRunning(cl, capiMachine)
				verifyMachinePausedCondition(capiMachine, mapiv1beta1.MachineAuthorityClusterAPI)

				By("Verifying there is a non-authoritative, paused MAPI Machine mirror for the new CAPI Machine")
				mapiMachine, err := mapiframework.GetLatestMachineFromMachineSet(ctx, cl, mapiMachineSet)
				Expect(err).ToNot(HaveOccurred(), "failed to get MAPI Machines from MachineSet")
				verifyMachineAuthoritative(mapiMachine, mapiv1beta1.MachineAuthorityClusterAPI)
				verifyMachinePausedCondition(mapiMachine, mapiv1beta1.MachineAuthorityClusterAPI)

				By("Deleting non-authoritative MAPI MachineSet")
				mapiMachineSet, err = mapiframework.GetMachineSet(ctx, cl, mapiMSAuthMAPIName)
				Expect(err).ToNot(HaveOccurred(), "failed to get mapiMachineSet")
				mapiframework.DeleteMachineSets(cl, mapiMachineSet)

				By("Verifying CAPI MachineSet not removed, both MAPI Machines and Mirrors remain")
				// TODO: Add full verification once OCPBUGS-56897 is fixed
				capiMachineSet = capiframework.GetMachineSet(cl, mapiMSAuthMAPIName, capiframework.CAPINamespace)
				Expect(capiMachineSet).ToNot(BeNil(), "CAPI MachineSet should still exist after deleting non-authoritative MAPI MachineSet")
				Expect(capiMachineSet.DeletionTimestamp.IsZero()).To(BeTrue(), "CAPI MachineSet should not be marked for deletion")
			})
		})
	})

	var _ = Describe("Field Conversion Tests", func() {
		var capiMSFieldTestName = "capi-field-conversion-test"
		var mapiMachineSet *mapiv1beta1.MachineSet
		var capiMachineSet *clusterv1.MachineSet
		var awsMachineTemplate *awsv1.AWSMachineTemplate

		BeforeAll(func() {
			mapiMachineSet = createMAPIMachineSetWithAuthoritativeAPI(ctx, cl, 0, capiMSFieldTestName, mapiv1beta1.MachineAuthorityClusterAPI, mapiv1beta1.MachineAuthorityClusterAPI)
			capiMachineSet, awsMachineTemplate = waitForMAPIMachineSetMirrors(cl, capiMSFieldTestName)

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

		Context("when CAPI MachineSet has all supported fields", func() {
			It("should convert AdditionalTags successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.AdditionalTags = awsv1.Tags{
						"Environment":                          "production",
						"Number":                          "123",
						"Special_Characters":                   "test@example.com:8080/path?query=value&foo=bar",
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) int {
						return len(p.Tags)
					},
					Equal(3),
					"Should have converted AdditionalTags to MAPI",
				)
			})

			It("should convert PlacementGroupName successfully", func() {
				testPlacementGroup := "prod-cluster-pg-partition-az1"
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.PlacementGroupName = testPlacementGroup
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) string {
						return p.PlacementGroupName
					},
					Equal(testPlacementGroup),
					"Should have converted PlacementGroupName to MAPI",
				)
			})

			It("should convert PlacementGroupPartition successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.PlacementGroupPartition = int64(5)
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) *int32 {
						return p.PlacementGroupPartition
					},
					HaveValue(Equal(int32(5))),
					"Should have converted PlacementGroupPartition to MAPI",
				)
			})

			It("should convert Tenancy successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.Tenancy = string(mapiv1beta1.DedicatedTenancy)
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) mapiv1beta1.InstanceTenancy {
						return p.Placement.Tenancy
					},
					Equal(mapiv1beta1.DedicatedTenancy),
					"Should have converted Tenancy to MAPI",
				)
			})

			It("should convert NetworkInterfaceType successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.NetworkInterfaceType = awsv1.NetworkInterfaceTypeEFAWithENAInterface
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) mapiv1beta1.AWSNetworkInterfaceType {
						return p.NetworkInterfaceType
					},
					Equal(mapiv1beta1.AWSEFANetworkInterfaceType),
					"Should have converted NetworkInterfaceType to MAPI",
				)
			})

			It("should convert SpotMarketOptions successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.SpotMarketOptions = &awsv1.SpotMarketOptions{
						MaxPrice: ptr.To("0.10"),
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) *mapiv1beta1.SpotMarketOptions {
						return p.SpotMarketOptions
					},
					Not(BeNil()),
					"Should have converted SpotMarketOptions to MAPI",
				)
			})

			It("should convert InstanceMetadataOptions successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.InstanceMetadataOptions = &awsv1.InstanceMetadataOptions{
						HTTPTokens: awsv1.HTTPTokensStateRequired,
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) mapiv1beta1.MetadataServiceAuthentication {
						return p.MetadataServiceOptions.Authentication
					},
					Equal(mapiv1beta1.MetadataServiceAuthenticationRequired),
					"Should have converted InstanceMetadataOptions to MAPI",
				)
			})

			It("should convert AdditionalSecurityGroups successfully", func() {
				testSGID := "sg-fedcba9876543210"
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.AdditionalSecurityGroups = []awsv1.AWSResourceReference{
						{ID: ptr.To(testSGID)},
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) int {
						return len(p.SecurityGroups)
					},
					Equal(1),
					"Should have converted AdditionalSecurityGroups to MAPI",
				)
			})

			It("should convert Subnet successfully", func() {
				testSubnetID := "subnet-fedcba9876543210"
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.Subnet = &awsv1.AWSResourceReference{
						ID: ptr.To(testSubnetID),
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) string {
						if p.Subnet.ID == nil {
							return ""
						}
						return *p.Subnet.ID
					},
					Equal(testSubnetID),
					"Should have converted Subnet to MAPI",
				)
			})

			It("should convert SSHKeyName successfully", func() {
				testKeyName := "openshift-prod-us-east-1-ssh-key"
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.SSHKeyName = ptr.To(testKeyName)
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) string {
						if p.KeyName == nil {
							return ""
						}
						return *p.KeyName
					},
					Equal(testKeyName),
					"Should have converted SSHKeyName to MAPI",
				)
			})

			It("should convert AMI successfully", func() {
				testAMIID := "ami-0a1b2c3d4e5f6g7h8"
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.AMI = awsv1.AMIReference{
						ID: ptr.To(testAMIID),
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) string {
						if p.AMI.ID == nil {
							return ""
						}
						return *p.AMI.ID
					},
					Equal(testAMIID),
					"Should have converted AMI to MAPI",
				)
			})

			It("should convert IAMInstanceProfile successfully", func() {
				testProfileName := "openshift-worker-node-instance-profile"
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.IAMInstanceProfile = testProfileName
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) string {
						if p.IAMInstanceProfile == nil || p.IAMInstanceProfile.ID == nil {
							return ""
						}
						return *p.IAMInstanceProfile.ID
					},
					Equal(testProfileName),
					"Should have converted IAMInstanceProfile to MAPI",
				)
			})

			It("should convert RootVolume successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.RootVolume = &awsv1.Volume{
						Size:       int64(500),
						Type:       awsv1.VolumeTypeGP3,
						IOPS:       int64(16000),
						Throughput: ptr.To(int64(1000)),
						Encrypted:  ptr.To(true),
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) int {
						return len(p.BlockDevices)
					},
					BeNumerically(">", 0),
					"Should have converted RootVolume to MAPI",
				)
			})

			It("should convert NonRootVolumes successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.NonRootVolumes = []awsv1.Volume{
						{
							DeviceName: "/dev/sdf",
							Size:       int64(2000),
							Type:       awsv1.VolumeTypeIO2,
							IOPS:       int64(64000),
							Encrypted:  ptr.To(true),
						},
					}
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) int {
						return len(p.BlockDevices)
					},
					BeNumerically(">", 0),
					"Should have converted NonRootVolumes to MAPI",
				)
			})

			It("should convert CapacityReservationID successfully", func() {
				testCapacityReservationID := "cr-fedcba9876543210"
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.CapacityReservationID = ptr.To(testCapacityReservationID)
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) string {
						return p.CapacityReservationID
					},
					Equal(testCapacityReservationID),
					"Should have converted CapacityReservationID to MAPI",
				)
			})

			It("should convert MarketType successfully", func() {
				awsMachineTemplate = updateCAPIMachineSetWithNewAWSMachineTemplate(ctx, cl, capiMachineSet, awsMachineTemplate, func(spec *awsv1.AWSMachineSpec) {
					spec.MarketType = awsv1.MarketTypeSpot
				})
				expectMAPIMachineSetProviderSpecField(ctx, cl, capiMSFieldTestName,
					func(p *mapiv1beta1.AWSMachineProviderConfig) mapiv1beta1.MarketType {
						return p.MarketType
					},
					Equal(mapiv1beta1.MarketTypeSpot),
					"Should have converted MarketType to MAPI",
				)
			})
		})
	})
})
