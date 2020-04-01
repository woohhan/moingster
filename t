#!/bin/bash

case "${1:-}" in
r)
  operator-sdk run --local
  ;;
g)
  operator-sdk generate crds
  operator-sdk generate k8s
  ;;
da)
  kubectl delete -f deploy/crds/moingster.com_v1alpha1_kluster_cr.yaml --ignore-not-found=true
  kubectl delete -f deploy/crds/moingster.com_klusters_crd.yaml --ignore-not-found=true
  ;;
dcr)
  kubectl delete -f deploy/crds/moingster.com_v1alpha1_kluster_cr.yaml --ignore-not-found=true
  ;;
dcrd)
  kubectl delete -f deploy/crds/moingster.com_klusters_crd.yaml --ignore-not-found=true
  ;;
aa)
  kubectl apply -f deploy/crds/moingster.com_klusters_crd.yaml
  kubectl apply -f deploy/crds/moingster.com_v1alpha1_kluster_cr.yaml
  ;;
acr)
  kubectl apply -f deploy/crds/moingster.com_v1alpha1_kluster_cr.yaml
  ;;
acrd)
  kubectl apply -f deploy/crds/moingster.com_klusters_crd.yaml
  ;;
*)
    echo " $0 [command]
Development Tools

Available Commands:
  r     Run local controller
  g     Generate code
  da    Delete All
  dcr   Delete CR
  dcrd  Delete CRD
  aa    Apply All
  acr   Apply CR
  acrd  Apply CRD
" >&2
    ;;
esac

