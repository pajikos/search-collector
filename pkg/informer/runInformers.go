// Copyright Contributors to the Open Cluster Management project

package informer

import (
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/stolostron/search-collector/pkg/config"
	rec "github.com/stolostron/search-collector/pkg/reconciler"
	tr "github.com/stolostron/search-collector/pkg/transforms"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Start and manages informers for resources in the cluster.
func RunInformers(initialized chan interface{}, upsertTransformer tr.Transformer, reconciler *rec.Reconciler) {

	// These functions return handler functions, which are then used in creation of the informers.
	createInformerAddHandler := func(resourceName string) func(interface{}) {
		return func(obj interface{}) {
			resource := obj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Create,
				Resource:       resource,
				ResourceString: resourceName,
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	createInformerUpdateHandler := func(resourceName string) func(interface{}, interface{}) {
		return func(oldObj, newObj interface{}) {
			resource := newObj.(*unstructured.Unstructured)
			upsert := tr.Event{
				Time:           time.Now().Unix(),
				Operation:      tr.Update,
				Resource:       resource,
				ResourceString: resourceName,
			}
			upsertTransformer.Input <- &upsert // Send resource into the transformer input channel
		}
	}

	informerDeleteHandler := func(obj interface{}) {
		resource := obj.(*unstructured.Unstructured)
		// We don't actually have anything to transform in the case of a deletion, so we manually construct the NodeEvent
		ne := tr.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: tr.Delete,
			Node: tr.Node{
				UID: strings.Join([]string{config.Cfg.ClusterName, string(resource.GetUID())}, "/"),
			},
		}
		reconciler.Input <- ne
	}

	// Get kubernetes client for discovering resource types
	discoveryClient := config.GetDiscoveryClient()

	// We keep each of the informer's stopper channel in a map, so we can stop them if the resource is no longer valid.
	stoppers := make(map[schema.GroupVersionResource]chan struct{})
	for {
		gvrList, err := SupportedResources(discoveryClient)
		if err != nil {
			glog.Error("Failed to get complete list of supported resources: ", err)
		}

		// Sometimes a partial list will be returned even if there is an error.
		// This could happen during install when a CRD hasn't fully initialized.
		if gvrList != nil {
			// Loop through the previous list of resources. If we find the entry in the new list we delete it so
			// that we don't end up with 2 informers. If we don't find it, we stop the informer that's currently
			// running because the resource no longer exists (or no longer supports watch).
			for gvr, stopper := range stoppers {
				// If this still exists in the new list, delete it from there as we don't want to recreate an informer
				if _, ok := gvrList[gvr]; ok {
					delete(gvrList, gvr)
					continue
				} else { // if it's in the old and NOT in the new, stop the informer
					glog.V(2).Infof("Resource %s no longer exists or no longer supports watch, stopping its informer\n", gvr.String())
					close(stopper)
					delete(stoppers, gvr)
				}
			}
			// Now, loop through the new list, which after the above deletions, contains only stuff that needs to
			// have a new informer created for it.
			for gvr := range gvrList {
				glog.V(2).Infof("Found new resource %s, creating informer\n", gvr.String())
				// Using our custom informer.
				informer, _ := InformerForResource(gvr)

				// Set up handler to pass this informer's resources into transformer
				informer.AddFunc = createInformerAddHandler(gvr.Resource)
				informer.UpdateFunc = createInformerUpdateHandler(gvr.Resource)
				informer.DeleteFunc = informerDeleteHandler

				stopper := make(chan struct{})
				stoppers[gvr] = stopper
				go informer.Run(stopper)
				// This wait serializes the informer initialization. It is needed to avoid a
				// spike in memory when the collector starts.
				informer.WaitUntilInitialized(time.Duration(10) * time.Second) // Times out after 10 seconds.
			}
			glog.V(2).Info("Total informers running: ", len(stoppers))

			initialized <- struct{}{}
		}

		time.Sleep(time.Duration(config.Cfg.RediscoverRateMS) * time.Millisecond)
	}
}
