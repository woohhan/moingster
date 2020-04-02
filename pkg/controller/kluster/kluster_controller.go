package kluster

import (
	"context"
	moingsterv1alpha1 "github.com/woohhan/moingster/pkg/apis/moingster/v1alpha1"
	"github.com/woohhan/moingster/pkg/kluster"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller_kluster")

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
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
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

func (r *ReconcileKluster) updateStatus(k *moingsterv1alpha1.Kluster, status moingsterv1alpha1.KlusterStatus) error {
	k.Status = status
	if err := r.client.Status().Update(context.TODO(), k); err != nil {
		return err
	}
	return nil
}

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

	// Delete first if we need
	if isKlusterTobeDeleted(k) {
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

	// First start
	if len(k.Status.State) == 0 {
		if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterCreating,}); err != nil {
			return reconcile.Result{}, err
		}

		// Add finalizer for this CR
		if !contains(k.GetFinalizers(), klusterFinalizer) {
			if err := r.addFinalizer(log, k); err != nil {
				return reconcile.Result{}, err
			}
		}

		// We doesn't return with `reconcile.Result{requeue: true}, nil` from here to end of this block
		// Because ssh secret, vm, disk are stateful, and doen't have any meaning to recreate.

		// Create ssh secret
		name := GetNamespacedName(k)
		secret, err := kluster.GetSshKeySecret(name)
		if err != nil {
			return reconcile.Result{}, err
		}
		if err := r.client.Create(context.TODO(), secret); err != nil {
			if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "SshSecretCreateFailed"}); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}

		// Create instance
		for i := 0; i < k.Spec.Nodes.Count; i++ {
			name := GetNamespacedNameWithIdx(k, i)

			// Create disk
			disk := kluster.GetDiskPvc(name, 6, "csi-hostpath-sc")
			reqLogger.Info("Create disk", "disk", disk)
			if err := r.client.Create(context.TODO(), disk); err != nil {
				if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "DiskPvcCreateFailed"}); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			}
			// TODO: Wait for disk preparing... Because hostpath has bug that make multiple disks once, It failed.
			err := wait.PollImmediate(3*time.Second, 300*time.Second, func() (done bool, err error) {
				pvc := &corev1.PersistentVolumeClaim{}
				if err := r.client.Get(context.TODO(), name, pvc); err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					} else {
						return false, err
					}
				}
				return pvc.Status.Phase == corev1.ClaimBound, nil
			})
			if err != nil {
				if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "DiskPvcCreateFailed"}); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			}

			// Create vm
			vm := kluster.GetVm(name, k.Name, 4096, 2, name.Name, string(secret.Data["publicKey"]))
			reqLogger.Info("Create vm", "vm", vm)
			if err := r.client.Create(context.TODO(), vm); err != nil {
				if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "VmCreateFailed"}); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			}

			// Create svc
			svc := kluster.GetSvc(name)
			reqLogger.Info("Create svc", "svc", svc)
			if err := r.client.Create(context.TODO(), svc); err != nil {
				if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "ServiceCreateFailed"}); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			}
		}

		// TODO: wait for ssh connection

		// Run kubespray job
		job := kluster.GetKubesprayJob(name);
		reqLogger.Info("Create job", "job", job)
		if err := r.client.Create(context.TODO(), job); err != nil {
			if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "JobCreateFailed"}); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		// Wait for job
		err = wait.Poll(5*time.Second, 60*time.Minute, func() (bool, error) {
			job := &batchv1.Job{}
			if err := r.client.Get(context.TODO(), name, job); err != nil {
				return false, err
			}
			if job.Status.Active > 0 {
				return false, nil
			}
			if job.Status.Succeeded > 0 {
				return true, nil
			}
			// still init...
			return false, nil
		})
		if err != nil {
			if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "JobWaitingFailed"}); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
		// Delete job
		if err := r.client.Delete(context.TODO(), job); err != nil {
			if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterError, Reason: "JobWaitingFailed"}); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}

		// Update to Created
		if err := r.updateStatus(k, moingsterv1alpha1.KlusterStatus{State: moingsterv1alpha1.KlusterAvailable}); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
