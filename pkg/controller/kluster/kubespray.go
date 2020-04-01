package kluster

import (
	"context"
	"fmt"
	moingsterv1alpha1 "github.com/woohhan/moingster/pkg/apis/moingster/v1alpha1"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

func (r *ReconcileKluster) kubespray(k *moingsterv1alpha1.Kluster) error {
	name := GetName(k)
	// first start, start kubespray
	if len(k.Status.State) == 0 {
		k.Status.State = "Creating"

		job := &v1.Job{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name.Name,
				Namespace: name.Namespace,
			},
			Spec: v1.JobSpec{
				BackoffLimit: nil,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "inventory",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/home/whhan91/.go/src/github.com/woohhan/moingster/docs/design/moingster_poc/inventory/",
									},
								},
							},
							{
								Name: "sshkey",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: name.Name,
										Items: []corev1.KeyToPath{
											{
												Key:  "privateKey",
												Path: "privateKey",
											},
										},
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:    "kubespray",
								Image:   "quay.io/woohhan/kubespray-docker:latest",
								Command: []string{"ansible-playbook", "-i", "/inventory/hosts.yaml", "--become", "--become-user=root", "cluster.yml", "--private-key=/ssh/privateKey"},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "inventory",
										MountPath: "/inventory",
									},
									{
										Name:      "sshkey",
										MountPath: "/ssh",
									},
								},
							},
						},
						RestartPolicy: "Never",
					},
				},
				TTLSecondsAfterFinished: nil,
			},
		}

		err := r.client.Create(context.TODO(), job)
		if err != nil {
			return err
		}

		// wait for job...
		err = wait.Poll(5*time.Second, 60*time.Minute, func() (bool, error) {
			job := &v1.Job{}
			err = r.client.Get(context.TODO(), types.NamespacedName{Name: k.Name + "-kubespray", Namespace: k.Namespace}, job)
			if err != nil {
				return false, err
			}

			if job.Status.Active > 0 {
				return false, nil
			}
			if job.Status.Failed > 0 {
				return false, fmt.Errorf("job failed")
			}
			if job.Status.Succeeded > 0 {
				return true, nil
			}
			// still init...
			return false, nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
