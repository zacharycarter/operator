package controller

import (
	core_util "github.com/appscode/kutil/core/v1"
	meta_util "github.com/appscode/kutil/meta"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs_util "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1/util"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) WaitUntilPaused(drmn *api.DormantDatabase) error {
	db := &api.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:      drmn.OffshootName(),
			Namespace: drmn.Namespace,
		},
	}

	if err := core_util.WaitUntilPodDeletedBySelector(c.Client, db.Namespace, metav1.SetAsLabelSelector(db.DeploymentLabels())); err != nil {
		return err
	}

	if err := core_util.WaitUntilServiceDeletedBySelector(c.Client, db.Namespace, metav1.SetAsLabelSelector(db.OffshootLabels())); err != nil {
		return err
	}

	return nil
}

func (c *Controller) deleteMatchingDormantDatabase(memcached *api.Memcached) error {
	// Check if DormantDatabase exists or not
	ddb, err := c.ExtClient.DormantDatabases(memcached.Namespace).Get(memcached.Name, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
		return nil
	}

	// Set WipeOut to false
	if _, _, err := cs_util.PatchDormantDatabase(c.ExtClient, ddb, func(in *api.DormantDatabase) *api.DormantDatabase {
		in.Spec.WipeOut = false
		return in
	}); err != nil {
		return err
	}

	// Delete  Matching dormantDatabase
	if err := c.ExtClient.DormantDatabases(memcached.Namespace).Delete(memcached.Name,
		meta_util.DeleteInBackground()); err != nil && !kerr.IsNotFound(err) {
		return err
	}

	return nil
}

func (c *Controller) createDormantDatabase(memcached *api.Memcached) (*api.DormantDatabase, error) {
	dormantDb := &api.DormantDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
			Labels: map[string]string{
				api.LabelDatabaseKind: api.ResourceKindMemcached,
			},
		},
		Spec: api.DormantDatabaseSpec{
			Origin: api.Origin{
				ObjectMeta: metav1.ObjectMeta{
					Name:              memcached.Name,
					Namespace:         memcached.Namespace,
					Labels:            memcached.Labels,
					Annotations:       memcached.Annotations,
					CreationTimestamp: memcached.CreationTimestamp,
				},
				Spec: api.OriginSpec{
					Memcached: &memcached.Spec,
				},
			},
		},
	}

	return c.ExtClient.DormantDatabases(dormantDb.Namespace).Create(dormantDb)
}
