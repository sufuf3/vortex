package kubernetes

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestGetNode(t *testing.T) {
	clientset := fakeclientset.NewSimpleClientset()

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "K8S-Node",
		},
	}
	_, err := clientset.CoreV1().Nodes().Create(&node)
	assert.NoError(t, err)

	result, err := GetNode(clientset, "K8S-Node")
	assert.NoError(t, err)
	assert.Equal(t, node.GetName(), result.GetName())
}

func TestGetNodeFail(t *testing.T) {
	clientset := fakeclientset.NewSimpleClientset()

	_, err := GetNode(clientset, "UnKnown_Name")
	assert.Error(t, err)
}

func TestGetNodes(t *testing.T) {
	clientset := fakeclientset.NewSimpleClientset()

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "K8S-Node1",
		},
	}
	_, err := clientset.CoreV1().Nodes().Create(&node)
	assert.NoError(t, err)

	node = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "K8S-Node2",
		},
	}
	_, err = clientset.CoreV1().Nodes().Create(&node)
	assert.NoError(t, err)

	nodes, err := GetNodes(clientset)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nodes))
}
