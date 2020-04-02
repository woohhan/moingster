package kluster

import (
	"fmt"
	moingsterv1alpha1 "github.com/woohhan/moingster/pkg/apis/moingster/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

func GetNamespacedName(k *moingsterv1alpha1.Kluster) types.NamespacedName {
	return types.NamespacedName{Name: fmt.Sprintf("kluster-%s", k.Name), Namespace: k.Namespace}
}

func GetNamespacedNameWithIdx(k *moingsterv1alpha1.Kluster, idx int) types.NamespacedName {
	return types.NamespacedName{Name: fmt.Sprintf("kluster-%s-%d", k.Name, idx), Namespace: k.Namespace}
}
