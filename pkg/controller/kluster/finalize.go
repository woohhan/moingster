package kluster

import (
	"context"
	"github.com/go-logr/logr"
	moingsterv1alpha1 "github.com/woohhan/moingster/pkg/apis/moingster/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubevirt "kubevirt.io/client-go/api/v1"
)

const klusterFinalizer = "finalizer.kluster.moingster.com"

func (r *ReconcileKluster) deleteIfExists(name types.NamespacedName, obj runtime.Object) error {
	if err := r.client.Get(context.TODO(), name, obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}

		return err
	}

	return r.client.Delete(context.TODO(), obj)
}

func (r *ReconcileKluster) finalizeKluster(reqLogger logr.Logger, k *moingsterv1alpha1.Kluster) error {
	for i := 0; i < k.Spec.Nodes.Count; i++ {
		name := GetNamespacedNameWithIdx(k, i)

		// Delete job
		job := &batchv1.Job{}
		reqLogger.Info("Delete job", "job", job)
		if err := r.deleteIfExists(name, job); err != nil {
			return err
		}

		// Delete vm
		vm := &kubevirt.VirtualMachine{}
		reqLogger.Info("Delete vm", "vm", vm)
		if err := r.deleteIfExists(name, vm); err != nil {
			return err
		}

		// Delete pvc
		pvc := &corev1.PersistentVolumeClaim{}
		reqLogger.Info("Delete pvc", "pvc", pvc)
		if err := r.deleteIfExists(name, pvc); err != nil {
			return err
		}

		// Delete svc
		svc := &corev1.Service{}
		reqLogger.Info("Delete svc", "svc", svc)
		if err := r.deleteIfExists(name, svc); err != nil {
			return err
		}
	}

	// Delete secret
	name := GetNamespacedName(k)
	secret := &corev1.Secret{}
	reqLogger.Info("Delete secret", "secret", secret)
	if err := r.deleteIfExists(name, secret); err != nil {
		return err
	}

	reqLogger.Info("Successfully finalized kluster")
	return nil
}

func isKlusterTobeDeleted(k *moingsterv1alpha1.Kluster) bool {
	return k.GetDeletionTimestamp() != nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (r *ReconcileKluster) addFinalizer(reqLogger logr.Logger, k *moingsterv1alpha1.Kluster) error {
	k.SetFinalizers(append(k.GetFinalizers(), klusterFinalizer))
	if err := r.client.Update(context.TODO(), k); err != nil {
		return err
	}
	return nil
}
