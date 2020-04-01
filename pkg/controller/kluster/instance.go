package kluster

import (
	"context"
	"fmt"
	moingsterv1alpha1 "github.com/woohhan/moingster/pkg/apis/moingster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirt "kubevirt.io/client-go/api/v1"
	"time"
)

type Instance struct {
	vm  *kubevirt.VirtualMachine
	svc *corev1.Service
}

func (i *Instance) String() string {
	return "hi"
}

// instanceReconcile create instance if not exists
func (r *ReconcileKluster) instanceReconcile(k *moingsterv1alpha1.Kluster, sshKey *corev1.Secret) ([]Instance, error) {
	ret := make([]Instance, k.Spec.Nodes.Count)
	for i := 0; i < k.Spec.Nodes.Count; i++ {
		var instance Instance

		name := GetIdxName(k, i)

		disk := &corev1.PersistentVolumeClaim{}
		sc := "csi-hostpath-sc"
		apiGroup := ""
		err := r.client.Get(context.TODO(), name, disk)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}

			disk = &corev1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name.Name,
					Namespace: name.Namespace,
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
					Resources: k8score.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": resource.MustParse("6Gi"),
						},
					},
					StorageClassName: &sc,
					DataSource: &k8score.TypedLocalObjectReference{
						Kind:     "PersistentVolumeClaim",
						Name:     "ubuntu",
						APIGroup: &apiGroup,
					},
				},
			}
			err = r.client.Create(context.TODO(), disk)
			if err != nil {
				return nil, err
			}

			// TODO:
			time.Sleep(10 * time.Second)
		}

		vm := &kubevirt.VirtualMachine{}
		err = r.client.Get(context.TODO(), name, vm)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}

			vm = newvm(name.Name, name.Namespace, name.Name, name.Name, 2, 4096, string(sshKey.Data["publicKey"]))
			err = r.client.Create(context.TODO(), vm)
			if err != nil {
				return nil, err
			}
		}

		svc := &corev1.Service{}
		err = r.client.Get(context.TODO(), name, svc)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}

			svc = &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name.Name,
					Namespace: name.Namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Protocol: "TCP",
							Port:     22,
						},
					},
					Selector: map[string]string{"kubevirt.io/domain": name.Name},
					Type:     "ClusterIP",
				},
			}
			err = r.client.Create(context.TODO(), svc)
			if err != nil {
				return nil, err
			}
		}

		instance.svc = svc
		ret = append(ret, instance)
	}

	return ret, nil
}

func newvm(name string, namespace string, klusterName string, disk string, cores int, memoryMb int, sshkey string) *kubevirt.VirtualMachine {
	running := true
	return &kubevirt.VirtualMachine{
		TypeMeta: k8smeta.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: "kubevirt.io/v1alpha3",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"kubevirt.io/os": "linux",
			},
		},
		Spec: kubevirt.VirtualMachineSpec{
			Running: &running,
			Template: &kubevirt.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: k8smeta.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/domain": name,
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
									ClaimName: disk,
								},
							},
						},
						{
							Name: "cloudinitdisk",
							VolumeSource: kubevirt.VolumeSource{
								CloudInitNoCloud: &kubevirt.CloudInitNoCloudSource{
									UserData: "#cloud-config\n" +
										"hostname: " + name + "\n" +
										"ssh_pwauth: True\n" +
										"disable_root: false\n" +
										"ssh_authorized_keys:\n" +
										"- " + sshkey,
								},
							},
						},
					},
				},
			},
		},
	}
}
