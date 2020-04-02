package kluster

import (
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetKubesprayJob(n types.NamespacedName) *v1.Job {
	return &v1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.Name,
			Namespace: n.Namespace,
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
									SecretName: n.Name,
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
							Name:  "kubespray",
							Image: "quay.io/woohhan/kubespray-docker:latest",
							Command: []string{"ansible-playbook", "-i", "/inventory/hosts.yaml", "--become",
								"--become-user=root", "cluster.yml", "--private-key=/ssh/privateKey"},
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
		},
	}
}
