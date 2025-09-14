package componenthelpers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	migrationsv1alpha1 "kubevirt.io/kubevirt-migration-controller/api/v1alpha1"
)

// Get a referenced MigCluster.
// Returns `nil` when the reference cannot be resolved.
func GetCluster(client k8sclient.Client, ref *corev1.ObjectReference) (*migrationsv1alpha1.MigCluster, error) {
	if ref == nil {
		return nil, nil
	}
	object := migrationsv1alpha1.MigCluster{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: ref.Namespace,
			Name:      ref.Name,
		},
		&object)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return &object, err
}
