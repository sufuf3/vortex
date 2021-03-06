package volume

import (
	"fmt"
	"strings"

	"github.com/linkernetworks/mongo"
	"github.com/linkernetworks/vortex/src/entity"
	"github.com/linkernetworks/vortex/src/kubeutils"
	"github.com/linkernetworks/vortex/src/serviceprovider"
	"gopkg.in/mgo.v2/bson"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getPVCInstance(volume *entity.Volume, name string, storageClassName string) *v1.PersistentVolumeClaim {
	capacity, _ := resource.ParseQuantity(volume.Capacity)
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{volume.AccessMode},
			Resources: v1.ResourceRequirements{
				Limits: map[v1.ResourceName]resource.Quantity{
					"storage": capacity,
				},
				Requests: map[v1.ResourceName]resource.Quantity{
					"storage": capacity,
				},
			},
			StorageClassName: &storageClassName,
		},
	}
}

func getStorageClassName(session *mongo.Session, storageName string) (string, error) {
	storage := entity.Storage{}
	err := session.FindOne(entity.StorageCollectionName, bson.M{"name": storageName}, &storage)
	return storage.StorageClassName, err
}

// CreateVolume is a function to create volume
func CreateVolume(sp *serviceprovider.Container, volume *entity.Volume) error {
	namespace := "default"
	session := sp.Mongo.NewSession()
	defer session.Close()
	//fetch the db to get the storageName
	storageName, err := getStorageClassName(session, volume.StorageName)
	if err != nil {
		return err
	}

	name := volume.GetPVCName()
	pvc := getPVCInstance(volume, name, storageName)
	_, err = sp.KubeCtl.CreatePVC(pvc, namespace)
	return err
}

// DeleteVolume is a function to delete volume
func DeleteVolume(sp *serviceprovider.Container, volume *entity.Volume) error {
	namespace := "default"
	//Check the pod
	session := sp.Mongo.NewSession()
	defer session.Close()

	pods, err := kubeutils.GetNonCompletedPods(sp, bson.M{"volumes.name": volume.Name})
	if err != nil {
		return err
	}

	usedPod := []string{}
	for _, pod := range pods {
		usedPod = append(usedPod, pod.Name)
	}
	if len(usedPod) != 0 {
		podNames := strings.Join(usedPod, ",")
		return fmt.Errorf("delete the volume [%s] fail, since the followings pods still ust it: %s", volume.Name, podNames)
	}

	return sp.KubeCtl.DeletePVC(volume.GetPVCName(), namespace)
}
