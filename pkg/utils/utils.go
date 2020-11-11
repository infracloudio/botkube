// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package utils

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	log "github.com/infracloudio/botkube/pkg/log"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// ResourceInformerMap is a map of resource name to resource Getter interface
	ResourceInformerMap map[string]cache.SharedIndexInformer
	// AllowedEventKindsMap is a map to filter valid event kinds
	AllowedEventKindsMap map[EventKind]bool
	// AllowedUpdateEventsMap is a map of resource and namespace to updateconfig
	AllowedUpdateEventsMap map[KindNS]config.UpdateSetting
	// AllowedKubectlResourceMap is map of allowed resources with kubectl command
	AllowedKubectlResourceMap map[string]bool
	// AllowedKubectlVerbMap is map of allowed verb with kubectl command
	AllowedKubectlVerbMap map[string]bool
	// KindResourceMap contains resource name to kind mapping
	KindResourceMap map[string]string
	// ShortnameResourceMap contains resource name to short name mapping
	ShortnameResourceMap map[string]string
	// DynamicKubeClient is a global dynamic kubernetes client to communicate to apiserver
	DynamicKubeClient dynamic.Interface
	// DynamicKubeInformerFactory is a global DynamicSharedInformerFactory object to watch resources
	DynamicKubeInformerFactory dynamicinformer.DynamicSharedInformerFactory
	// Mapper is a global DeferredDiscoveryRESTMapper object, which maps all resources present on
	// the cluster, and create relation between GVR, and GVK
	Mapper *restmapper.DeferredDiscoveryRESTMapper
	// DiscoveryClient implements
	DiscoveryClient discovery.DiscoveryInterface
)

// InitKubeClient creates K8s client from provided kubeconfig OR service account to interact with apiserver
func InitKubeClient() {
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
		}
		botkubeConf, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Fatal(err)
		}
		// Initiate discovery client for REST resource mapping
		DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(botkubeConf)
		if err != nil {
			log.Fatalf("Unable to create Discovery Client")
		}
		DynamicKubeClient, err = dynamic.NewForConfig(botkubeConf)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// Initiate discovery client for REST resource mapping
		DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(kubeConfig)
		if err != nil {
			log.Fatal(err)
		}
		DynamicKubeClient, err = dynamic.NewForConfig(kubeConfig)
		if err != nil {
			log.Fatal(err)
		}
	}

	discoCacheClient := cacheddiscovery.NewMemCacheClient(DiscoveryClient)
	discoCacheClient.Invalidate()
	Mapper = restmapper.NewDeferredDiscoveryRESTMapper(discoCacheClient)

}

// EventKind used in AllowedEventKindsMap to filter event kinds
type EventKind struct {
	Resource  string
	Namespace string
	EventType config.EventType
}

// KindNS used in AllowedUpdateEventsMap
type KindNS struct {
	Resource  string
	Namespace string
}

// InitInformerMap initializes helper maps to filter events
func InitInformerMap(conf *config.Config) {
	// Get resync period
	rsyncTimeStr, ok := os.LookupEnv("INFORMERS_RESYNC_PERIOD")
	if !ok {
		rsyncTimeStr = "30"
	}
	rsyncTime, err := strconv.Atoi(rsyncTimeStr)
	if err != nil {
		log.Fatal("Error in reading INFORMERS_RESYNC_PERIOD env var.", err)
	}

	// Create dynamic shared informer factory
	DynamicKubeInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(DynamicKubeClient, time.Duration(rsyncTime)*time.Minute)

	// Init maps
	ResourceInformerMap = make(map[string]cache.SharedIndexInformer)
	AllowedEventKindsMap = make(map[EventKind]bool)
	AllowedUpdateEventsMap = make(map[KindNS]config.UpdateSetting)

	for _, v := range conf.Resources {
		gvr, err := ParseResourceArg(v.Name)
		if err != nil {
			log.Infof("Unable to parse resource: %v\n", v.Name)
			continue
		}

		ResourceInformerMap[v.Name] = DynamicKubeInformerFactory.ForResource(gvr).Informer()
	}
	// Allowed event kinds map and Allowed Update Events Map
	for _, r := range conf.Resources {
		allEvents := false
		for _, e := range r.Events {
			if e == config.AllEvent {
				allEvents = true
				break
			}
			for _, ns := range r.Namespaces.Include {
				AllowedEventKindsMap[EventKind{Resource: r.Name, Namespace: ns, EventType: e}] = true
			}
			// AllowedUpdateEventsMap entry is created only for UpdateEvent
			if e == config.UpdateEvent {
				for _, ns := range r.Namespaces.Include {
					AllowedUpdateEventsMap[KindNS{Resource: r.Name, Namespace: ns}] = r.UpdateSetting
				}
			}
		}

		// For AllEvent type, add all events to map
		if allEvents {
			events := []config.EventType{config.CreateEvent, config.UpdateEvent, config.DeleteEvent, config.ErrorEvent}
			for _, ev := range events {
				for _, ns := range r.Namespaces.Include {
					AllowedEventKindsMap[EventKind{Resource: r.Name, Namespace: ns, EventType: ev}] = true
					AllowedUpdateEventsMap[KindNS{Resource: r.Name, Namespace: ns}] = r.UpdateSetting
				}
			}
		}
	}
	log.Infof("Allowed Events - %+v", AllowedEventKindsMap)
	log.Infof("Allowed UpdateEvents - %+v", AllowedUpdateEventsMap)
}

// GetObjectMetaData returns metadata of the given object
func GetObjectMetaData(obj interface{}) metaV1.ObjectMeta {
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return metaV1.ObjectMeta{}
	}
	unstructuredObject = unstructuredObject.DeepCopy()
	objectMeta := metaV1.ObjectMeta{
		Name:                       unstructuredObject.GetName(),
		GenerateName:               unstructuredObject.GetGenerateName(),
		Namespace:                  unstructuredObject.GetNamespace(),
		ResourceVersion:            unstructuredObject.GetResourceVersion(),
		Generation:                 unstructuredObject.GetGeneration(),
		CreationTimestamp:          unstructuredObject.GetCreationTimestamp(),
		DeletionTimestamp:          unstructuredObject.GetDeletionTimestamp(),
		DeletionGracePeriodSeconds: unstructuredObject.GetDeletionGracePeriodSeconds(),
		Labels:                     unstructuredObject.GetLabels(),
		Annotations:                unstructuredObject.GetAnnotations(),
		OwnerReferences:            unstructuredObject.GetOwnerReferences(),
		Finalizers:                 unstructuredObject.GetFinalizers(),
		ClusterName:                unstructuredObject.GetClusterName(),
		ManagedFields:              unstructuredObject.GetManagedFields(),
	}
	if GetObjectTypeMetaData(obj).Kind == "Event" {
		var eventObj coreV1.Event
		err := TransformIntoTypedObject(obj.(*unstructured.Unstructured), &eventObj)
		if err != nil {
			log.Errorf("Unable to transform object type: %v, into type: %v", reflect.TypeOf(obj), reflect.TypeOf(eventObj))
		}
		if len(objectMeta.Annotations) == 0 {
			objectMeta.Annotations = ExtractAnnotationsFromEvent(&eventObj)
		} else {
			// Append InvolvedObject`s annotations to existing event object`s annotations map
			for key, value := range ExtractAnnotationsFromEvent(&eventObj) {
				objectMeta.Annotations[key] = value
			}
		}
	}
	return objectMeta
}

// GetObjectTypeMetaData returns typemetadata of the given object
func GetObjectTypeMetaData(obj interface{}) metaV1.TypeMeta {

	k, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return metaV1.TypeMeta{}
	}
	k = k.DeepCopy()
	return metaV1.TypeMeta{
		APIVersion: k.GetAPIVersion(),
		Kind:       k.GetKind(),
	}
}

// DeleteDoubleWhiteSpace returns slice that removing whitespace from a arg slice
func DeleteDoubleWhiteSpace(slice []string) []string {
	result := []string{}
	for _, s := range slice {
		if len(s) != 0 {
			result = append(result, s)
		}
	}
	return result
}

// GetResourceFromKind returns resource name for given Kind
func GetResourceFromKind(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	mapping, err := Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("Error while creating REST Mapping for Event Involved Object: %v", err)
	}
	return mapping.Resource, nil
}

// ExtractAnnotationsFromEvent returns annotations of InvolvedObject for the given event
func ExtractAnnotationsFromEvent(obj *coreV1.Event) map[string]string {
	gvr, err := GetResourceFromKind(obj.InvolvedObject.GroupVersionKind())
	if err != nil {
		log.Error(err)
		return nil
	}
	annotations, err := DynamicKubeClient.Resource(gvr).Namespace(obj.InvolvedObject.Namespace).Get(obj.InvolvedObject.Name, metaV1.GetOptions{})
	if err != nil {
		log.Error(err)
		return nil
	}
	return annotations.GetAnnotations()
}

// InitResourceMap initializes helper maps to allow kubectl execution for required resources
func InitResourceMap(conf *config.Config) {
	if !conf.Settings.Kubectl.Enabled {
		return
	}
	KindResourceMap = make(map[string]string)
	ShortnameResourceMap = make(map[string]string)
	AllowedKubectlResourceMap = make(map[string]bool)
	AllowedKubectlVerbMap = make(map[string]bool)

	for _, r := range conf.Settings.Kubectl.Commands.Resources {
		AllowedKubectlResourceMap[r] = true
	}

	for _, r := range conf.Settings.Kubectl.Commands.Verbs {
		AllowedKubectlVerbMap[r] = true
	}

	resourceList, err := DiscoveryClient.ServerResources()
	if err != nil {
		log.Errorf("Failed to get resource list in k8s cluster. %v", err)
		return
	}
	for _, resource := range resourceList {
		for _, r := range resource.APIResources {
			// Ignore subresources
			if strings.Contains(r.Name, "/") {
				continue
			}
			KindResourceMap[strings.ToLower(r.Kind)] = r.Name
			for _, sn := range r.ShortNames {
				ShortnameResourceMap[sn] = r.Name
			}
		}
	}
	log.Infof("AllowedKubectlResourceMap - %+v", AllowedKubectlResourceMap)
	log.Infof("AllowedKubectlVerbMap - %+v", AllowedKubectlVerbMap)
	log.Infof("KindResourceMap - %+v", KindResourceMap)
	log.Infof("ShortnameResourceMap - %+v", ShortnameResourceMap)
}

//GetClusterNameFromKubectlCmd this will return cluster name from kubectl command
func GetClusterNameFromKubectlCmd(cmd string) string {
	r, _ := regexp.Compile(`--cluster-name[=|' ']([^\s]*)`)
	//this gives 2 match with cluster name and without
	matchedArray := r.FindStringSubmatch(cmd)
	s := ""
	if len(matchedArray) >= 2 {
		s = matchedArray[1]
	}
	return s
}

// ParseResourceArg parses the group/version/resource args and create a schema.GroupVersionResource
func ParseResourceArg(arg string) (schema.GroupVersionResource, error) {
	var gvr schema.GroupVersionResource
	if strings.Count(arg, "/") >= 2 {
		s := strings.SplitN(arg, "/", 3)
		gvr = schema.GroupVersionResource{Group: s[0], Version: s[1], Resource: s[2]}
	} else if strings.Count(arg, "/") == 1 {
		s := strings.SplitN(arg, "/", 2)
		gvr = schema.GroupVersionResource{Group: "", Version: s[0], Resource: s[1]}
	}

	// Validate the GVR provided
	if _, err := Mapper.ResourcesFor(gvr); err != nil {
		return schema.GroupVersionResource{}, err
	}
	return gvr, nil
}

// GVRToString converts GVR formats to string
func GVRToString(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return fmt.Sprintf("%s/%s", gvr.Version, gvr.Resource)
	}
	return fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
}

// TransformIntoTypedObject uses unstructured interface and creates a typed object
func TransformIntoTypedObject(obj *unstructured.Unstructured, typedObject interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObject)
}

//GetStringInYamlFormat get the formatted commands list
func GetStringInYamlFormat(header string, commands map[string]bool) string {
	var b bytes.Buffer
	fmt.Fprintln(&b, header)
	for k, v := range commands {
		if v {
			fmt.Fprintf(&b, "  - %s\n", k)
		}
	}
	return b.String()
}

// CheckOperationAllowed checks whether operation are allowed
func CheckOperationAllowed(eventMap map[EventKind]bool, namespace string, resource string, eventType config.EventType) bool {
	if eventMap != nil && (eventMap[EventKind{
		Resource:  resource,
		Namespace: "all",
		EventType: eventType}] ||
		eventMap[EventKind{
			Resource:  resource,
			Namespace: namespace,
			EventType: eventType}]) {
		return true
	}
	return false
}

// Contains tells whether a contains x.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if strings.ToLower(x) == strings.ToLower(n) {
			return true
		}
	}
	return false
}

// GetAllChannels return all channels listed in Accessbinding
func GetAllChannels(accessBindings *[]config.AccessBinding) []string {
	allChannels := []string{}
	for _, accessBind := range *accessBindings {
		allChannels = append(allChannels, accessBind.ChannelName)
	}
	return allChannels
}

// FormatProfile will remove the unnecessary part of the message, while displaying profile information
func FormatProfile(profile *string) error {
	profileValue := *profile
	// Remove the last two default key value from the each profile value (Kubectl struct)
	eliminate1 := `defaultnamespace: ""`
	eliminate2 := `restrictAccess: false`
	profileValue = strings.ReplaceAll(profileValue, eliminate1, "")
	profileValue = strings.ReplaceAll(profileValue, eliminate2, "")
	*profile = profileValue
	return nil
}
