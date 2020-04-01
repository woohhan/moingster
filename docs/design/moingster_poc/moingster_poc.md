# Moingster POC
- Kubevirt에 의해 VM 3개 생성
- Kubespray Pod을 생성해서 해당 VM에 k8s를 인스톨
- 외부에서 해당 k8s에 접근할 수 있도록 expose

## 설치
```shell
# Kubevirt 설치
export KUBEVIRT_VERSION=v0.27.0
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml

# CDI 설치
export VERSION=v1.13.2
kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-operator.yaml
kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-cr.yaml

# hostpath csi 설치
SNAPSHOTTER_VERSION=v2.0.1
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml
CSI_ATTACHER_TAG=v2.1.1 ./csi-driver-host-path/deploy/kubernetes-1.17/deploy-hostpath.sh     # attacher v2.1.0 이하에서 disk를 여러개 한 번에 만들면 안만들어짐
kubectl apply -f ./csi-driver-host-path/examples/csi-storageclass.yaml

# 이미지 업로드
kubectl apply -f upload-proxy.yaml
kubectl apply -f ubuntu.yaml
TOKEN=$(kubectl apply -f upload-token.yaml -o="jsonpath={.status.token}")
curl -v --insecure -H "Authorization: Bearer $TOKEN" --data-binary @/home/whhan91/imgs/bionic-server-cloudimg-amd64.img https://$(sudo minikube ip):31001/v1alpha1/upload

# 각 vm을 위한 pvc 생성
kubectl apply -f disk0.yaml
kubectl apply -f disk1.yaml
kubectl apply -f disk2.yaml

# VM과 서비스 생성
kubectl apply -f vm0.yaml
kubectl apply -f vm1.yaml
kubectl apply -f vm2.yaml

# kubespray 실행과 서비스 생성
kubectl apply -f install.yaml

# kubeconfig 셋업
# admin.conf의 서버 주소를 노드포트 주소로 변경
```

## 제거
```shell
kubectl delete -f install.yaml

# clean VM
kubectl delete -f vm2.yaml
kubectl delete -f vm1.yaml
kubectl delete -f vm0.yaml
kubectl delete -f disk2.yaml
kubectl delete -f disk1.yaml
kubectl delete -f disk0.yaml

# clean ubuntu image
kubectl delete -f pvc_ubuntu.yaml
```
