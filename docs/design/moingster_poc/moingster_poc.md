# Moingster POC
- Kubevirt에 의해 VM 3개 생성
- Kubespray Pod을 생성해서 해당 VM에 k8s를 인스톨
- 외부에서 해당 k8s에 접근할 수 있도록 expose

## 설치
```shell
export KUBEVIRT_VERSION=v0.27.0
kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml
kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml

export VERSION=v1.13.2
kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-operator.yaml
kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-cr.yaml

# HTTP로부터 ubuntu 이미지 클론
kubectl apply -f pvc_ubuntu.yaml

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
