package kluster

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubevirt "kubevirt.io/client-go/api/v1"
)

func GetDiskPvc(n types.NamespacedName, sizeGi int, scName string) *corev1.PersistentVolumeClaim {
	apiGroup := "" // TODO: need?
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.Name,
			Namespace: n.Namespace,
		},
		Spec: k8score.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: k8score.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": resource.MustParse(fmt.Sprintf("%dGi", sizeGi)),
				},
			},
			StorageClassName: &scName,
			DataSource: &k8score.TypedLocalObjectReference{
				Kind:     "PersistentVolumeClaim",
				Name:     "ubuntu",
				APIGroup: &apiGroup,
			},
		},
	}
}

func GetVm(n types.NamespacedName, klusterName string, memoryMb int, cores int, diskPvcName string, sshPublicKey string) *kubevirt.VirtualMachine {
	running := true
	return &kubevirt.VirtualMachine{
		TypeMeta: k8smeta.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: "kubevirt.io/v1alpha3",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:      n.Name,
			Namespace: n.Namespace,
			Labels: map[string]string{
				"kubevirt.io/os": "linux",
			},
		},
		Spec: kubevirt.VirtualMachineSpec{
			Running: &running,
			Template: &kubevirt.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/domain": n.Name,
						"kluster":            klusterName,
					},
				},
				Spec: kubevirt.VirtualMachineInstanceSpec{
					Domain: kubevirt.DomainSpec{
						Resources: kubevirt.ResourceRequirements{
							Requests: map[k8score.ResourceName]resource.Quantity{
								"memory": resource.MustParse(fmt.Sprintf("%dM", memoryMb)),
							},
						},
						CPU: &kubevirt.CPU{
							Cores: uint32(cores),
						},
						Machine: kubevirt.Machine{
							Type: "q35",
						},
						Devices: kubevirt.Devices{
							Disks: []kubevirt.Disk{
								{
									Name: "root",
									DiskDevice: kubevirt.DiskDevice{
										Disk: &kubevirt.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{
									Name: "cloudinitdisk",
									DiskDevice: kubevirt.DiskDevice{
										CDRom: &kubevirt.CDRomTarget{
											Bus: "sata",
										},
									},
								},
							},
						},
					},
					Volumes: []kubevirt.Volume{
						{
							Name: "root",
							VolumeSource: kubevirt.VolumeSource{
								PersistentVolumeClaim: &k8score.PersistentVolumeClaimVolumeSource{
									ClaimName: diskPvcName,
								},
							},
						},
						{
							Name: "cloudinitdisk",
							VolumeSource: kubevirt.VolumeSource{
								CloudInitNoCloud: &kubevirt.CloudInitNoCloudSource{
									UserData: "#cloud-config\n" +
										"hostname: " + n.Name + "\n" +
										"ssh_pwauth: True\n" +
										"disable_root: false\n" +
										"ssh_authorized_keys:\n" +
										"- " + sshPublicKey,
								},
							},
						},
					},
				},
			},
		},
	}
}

func GetSvc(n types.NamespacedName) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.Name,
			Namespace: n.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: "TCP",
					Port:     22,
				},
			},
			Selector: map[string]string{"kubevirt.io/domain": n.Name},
			Type:     "ClusterIP",
		},
	}
}
