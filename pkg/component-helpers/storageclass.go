package componenthelpers

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	migrationsv1alpha1 "kubevirt.io/kubevirt-migration-controller/api/v1alpha1"
)

// Get the list StorageClasses in the format expected by PV discovery
func GetStorageClasses(client k8sclient.Client) ([]migrationsv1alpha1.StorageClass, error) {
	list := storagev1.StorageClassList{}
	err := client.List(context.TODO(), &list, &k8sclient.ListOptions{})
	if err != nil {
		return nil, err
	}
	kubeStorageClasses := list.Items
	// Transform kube storage classes into format used in PV discovery
	var storageClasses []migrationsv1alpha1.StorageClass
	for _, clusterStorageClass := range kubeStorageClasses {
		blockAccessModes := accessModesForProvisioner(clusterStorageClass.Provisioner, corev1.PersistentVolumeBlock)
		storageClass := migrationsv1alpha1.StorageClass{
			Name:        clusterStorageClass.Name,
			Provisioner: clusterStorageClass.Provisioner,
			VolumeAccessModes: []migrationsv1alpha1.VolumeModeAccessMode{
				{
					VolumeMode:  corev1.PersistentVolumeFilesystem,
					AccessModes: accessModesForProvisioner(clusterStorageClass.Provisioner, corev1.PersistentVolumeFilesystem),
				},
			},
		}
		if len(blockAccessModes) > 0 {
			storageClass.VolumeAccessModes = append(storageClass.VolumeAccessModes, migrationsv1alpha1.VolumeModeAccessMode{
				VolumeMode:  corev1.PersistentVolumeBlock,
				AccessModes: blockAccessModes,
			})
		}
		if clusterStorageClass.Annotations != nil {
			isDefault, _ := strconv.ParseBool(clusterStorageClass.Annotations["storageclass.kubernetes.io/is-default-class"])
			if isDefault {
				storageClass.Default = true
			} else {
				storageClass.Default, _ = strconv.ParseBool(clusterStorageClass.Annotations["storageclass.beta.kubernetes.io/is-default-class"])
			}
		}
		storageClasses = append(storageClasses, storageClass)
	}
	return storageClasses, nil
}

// Gets the list of supported access modes for a provisioner
// TODO: allow the in-file mapping to be overridden by a configmap
func accessModesForProvisioner(provisioner string, volumeMode corev1.PersistentVolumeMode) []corev1.PersistentVolumeAccessMode {
	// TODO: implement via storage profiles instead of originally hardcoded values in MTC

	if volumeMode == corev1.PersistentVolumeBlock {
		return nil
	}
	// default value
	return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
}
