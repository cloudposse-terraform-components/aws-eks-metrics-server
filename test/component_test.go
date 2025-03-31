package test

import (
	"context"
	"fmt"
	"testing"
	"strings"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	awsHelper "github.com/cloudposse/test-helpers/pkg/aws"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	"github.com/cloudposse/test-helpers/pkg/helm"
	"github.com/stretchr/testify/assert"
	"github.com/gruntwork-io/terratest/modules/random"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	aggregatorclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type ComponentSuite struct {
	helper.TestSuite
}

func (s *ComponentSuite) TestBasic() {
	const component = "eks/metrics-server/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	randomID := strings.ToLower(random.UniqueId())

	namespace := fmt.Sprintf("metrics-server-%s", randomID)

	inputs := map[string]interface{}{
		"kubernetes_namespace": namespace,
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	metadataArray := []helm.Metadata{}

	atmos.OutputStruct(s.T(), options, "metadata", &metadataArray)

	assert.Equal(s.T(), len(metadataArray), 1)
	metadata := metadataArray[0]

	assert.Equal(s.T(), metadata.AppVersion, "0.6.2")
	assert.Equal(s.T(), metadata.Chart, "metrics-server")
	assert.NotNil(s.T(), metadata.FirstDeployed)
	assert.NotNil(s.T(), metadata.LastDeployed)
	assert.Equal(s.T(), metadata.Name, "metrics-server")
	assert.Equal(s.T(), metadata.Namespace, namespace)
	assert.NotNil(s.T(), metadata.Values)
	assert.Equal(s.T(), metadata.Version, "6.2.6")

	clusterOptions := s.GetAtmosOptions("eks/cluster", stack, nil)
	clusrerId := atmos.Output(s.T(), clusterOptions, "eks_cluster_id")

	cluster := awsHelper.GetEksCluster(s.T(), context.Background(), awsRegion, clusrerId)


	config, err := awsHelper.NewK8SClientConfig(cluster)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), config)

	clientset, err := aggregatorclientset.NewForConfig(config)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), clientset)

	// Retrieve the APIService for v1beta1.metrics.k8s.io
	apiService, err := clientset.ApiregistrationV1().APIServices().Get(context.Background(), "v1beta1.metrics.k8s.io", metav1.GetOptions{})
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), apiService.Spec.Service)

	assert.Equal(s.T(), apiService.Spec.Service.Name, "metrics-server")
	assert.Equal(s.T(), apiService.Spec.Service.Namespace, namespace)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "eks/metrics-server/disabled"
	const stack = "default-test"
	s.VerifyEnabledFlag(component, stack, nil)
}

func (s *ComponentSuite) SetupSuite() {
	s.TestSuite.InitConfig()
	s.TestSuite.Config.ComponentDestDir = "components/terraform/eks/metrics-server"
	s.TestSuite.SetupSuite()
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	suite.AddDependency(t, "vpc", "default-test", nil)
	suite.AddDependency(t, "eks/cluster", "default-test", nil)
	helper.Run(t, suite)
}
