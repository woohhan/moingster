package kluster

import (
	"context"
	"github.com/go-logr/logr"
	moingsterv1alpha1 "github.com/woohhan/moingster/pkg/apis/moingster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirt "kubevirt.io/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kluster")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Kluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Kluster
	err = c.Watch(&source.Kind{Type: &moingsterv1alpha1.Kluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Kluster
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &moingsterv1alpha1.Kluster{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKluster{}

// ReconcileKluster reconciles a Kluster object
type ReconcileKluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

const klusterFinalizer = "finalizer.kluster.moingster.com"

// Reconcile reads that state of the cluster for a Kluster object and makes changes based on the state read
func (r *ReconcileKluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Kluster")

	// Fetch the Kluster
	k := &moingsterv1alpha1.Kluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, k)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	isKlusterToBeDeleted := k.GetDeletionTimestamp() != nil
	if isKlusterToBeDeleted {
		if contains(k.GetFinalizers(), klusterFinalizer) {
			if err := r.finalizeKluster(reqLogger, k); err != nil {
				return reconcile.Result{}, err
			}

			k.SetFinalizers(remove(k.GetFinalizers(), klusterFinalizer))
			if err := r.client.Update(context.TODO(), k); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(k.GetFinalizers(), klusterFinalizer) {
		if err := r.addFinalizer(log, k); err != nil {
			return reconcile.Result{}, err
		}
	}

	secret, err := r.secretReconcile(k)
	if err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.Info("secretReconcile OK", "secret", secret)

	instances, err := r.instanceReconcile(k, secret)
	if err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.Info("instanceReconcile OK", "instance", instances)

	/*
		err = r.kubespray(k)
		if err != nil {
			return reconcile.Result{}, err
		}*/

	return reconcile.Result{}, nil
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

func (r *ReconcileKluster) finalizeKluster(reqLogger logr.Logger, k *moingsterv1alpha1.Kluster) error {
	for i := 0; i < k.Spec.Nodes.Count; i++ {
		name := GetIdxName(k, i)

		// Delete vm
		vm := &kubevirt.VirtualMachine{}
		if err := r.client.Get(context.TODO(), name, vm); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		if err := r.client.Delete(context.TODO(), vm); err != nil {
			return err
		}

		// Delete pvc
		pvc := &corev1.PersistentVolumeClaim{}
		if err := r.client.Get(context.TODO(), name, pvc); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		if err := r.client.Delete(context.TODO(), pvc); err != nil {
			return err
		}

		// Delete svc
		svc := &corev1.Service{}
		if err := r.client.Get(context.TODO(), name, svc); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		if err := r.client.Delete(context.TODO(), svc); err != nil {
			return err
		}
	}

	// Delete secret
	name := GetName(k)
	secret := &corev1.Secret{}
	if err := r.client.Get(context.TODO(), name, secret); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	if err := r.client.Delete(context.TODO(), secret); err != nil {
		return err
	}

	reqLogger.Info("Successfully finalized kluster")
	return nil
}
