package util

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1"
	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/wait"
)

func CreateOrPatchEtcd(c cs.KubedbV1alpha1Interface, meta metav1.ObjectMeta, transform func(*api.Etcd) *api.Etcd) (*api.Etcd, kutil.VerbType, error) {
	cur, err := c.Etcds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Etcd %s/%s.", meta.Namespace, meta.Name)
		out, err := c.Etcds(meta.Namespace).Create(transform(&api.Etcd{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Etcd",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchEtcd(c, cur, transform)
}

func PatchEtcd(c cs.KubedbV1alpha1Interface, cur *api.Etcd, transform func(*api.Etcd) *api.Etcd) (*api.Etcd, kutil.VerbType, error) {
	return PatchEtcdObject(c, cur, transform(cur.DeepCopy()))
}

func PatchEtcdObject(c cs.KubedbV1alpha1Interface, cur, mod *api.Etcd) (*api.Etcd, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(curJson, modJson, curJson)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Etcd %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.Etcds(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateEtcd(c cs.KubedbV1alpha1Interface, meta metav1.ObjectMeta, transform func(*api.Etcd) *api.Etcd) (result *api.Etcd, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.Etcds(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {

			result, e2 = c.Etcds(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Etcd %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Etcd %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateEtcdStatus(c cs.KubedbV1alpha1Interface, cur *api.Etcd, transform func(*api.EtcdStatus) *api.EtcdStatus, useSubresource ...bool) (*api.Etcd, error) {
	if len(useSubresource) > 1 {
		return nil, errors.Errorf("invalid value passed for useSubresource: %v", useSubresource)
	}

	mod := &api.Etcd{
		TypeMeta:   cur.TypeMeta,
		ObjectMeta: cur.ObjectMeta,
		Spec:       cur.Spec,
		Status:     *transform(cur.Status.DeepCopy()),
	}

	if len(useSubresource) == 1 && useSubresource[0] {
		return c.Etcds(cur.Namespace).UpdateStatus(mod)
	}

	out, _, err := PatchEtcdObject(c, cur, mod)
	return out, err
}
