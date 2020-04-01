package kluster

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	moingsterv1alpha1 "github.com/woohhan/moingster/pkg/apis/moingster/v1alpha1"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretReconcile creates ssh key secret if not exists
func (r *ReconcileKluster) secretReconcile(k *moingsterv1alpha1.Kluster) (*corev1.Secret, error) {
	name := GetName(k)
	secret := &corev1.Secret{}
	err := r.client.Get(context.TODO(), name, secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}

		publicKey, privateKey, err := genSshKeyPair()
		if err != nil {
			return nil, err
		}

		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name.Name,
				Namespace: name.Namespace,
			},
			StringData: map[string]string{"privateKey": privateKey, "publicKey": publicKey},
			Type:       "Opaque",
		}
		err = r.client.Create(context.TODO(), secret)
		if err != nil {
			return nil, err
		}
	}
	return secret, nil
}

func genSshKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	var private bytes.Buffer
	if err := pem.Encode(&private, privateKeyPEM); err != nil {
		return "", "", err
	}

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	public := ssh.MarshalAuthorizedKey(pub)
	return string(public), private.String(), nil
}
