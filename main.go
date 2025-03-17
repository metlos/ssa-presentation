package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprint(os.Stderr, "Pass the name of the namespace and the number of the slide as the arguments\n")
		os.Exit(1)
	}

	ns := os.Args[1]

	slideNumber, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse the slide number: %s", err.Error())
		os.Exit(1)
	}

	cl := getClient()

	nsObj := object("", `
apiVersion: v1
kind: Namespace
metadata:
    name: `+ns+`
`)
	deleteIfExist(cl, nsObj)

	exitIfError(cl.Create(context.TODO(), nsObj))

	slideFns := []func(client.Client, string) error{
		slide1, slide2, slide3, slide4, slide5, slide6, slide7, slide8, slide9, slide10,
	}
	for i := 0; i < slideNumber; i++ {
		if err := slideFns[i](cl, ns); err != nil && i == slideNumber-1 {
			fmt.Fprintf(os.Stderr, "slide %d failed: %s\n", i+1, err.Error())
			os.Exit(1)
		}
	}
}

func slide1(cl client.Client, ns string) error {
	// nothing to be done here
	return nil
}

func slide2(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("alice"))
}

func slide3(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  my-key: my-value
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("bob"))
}

func slide4(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: my-value
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("bob"))
}

func slide5(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
  my-key: my-value
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("bob"))
}

func slide6(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: my-value
  my-key: my-value
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("bob"), client.ForceOwnership)
}

func slide7(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("bob"), client.ForceOwnership)
}

func slide8(cl client.Client, ns string) error {
	obj := &corev1.ConfigMap{}
	if err := cl.Get(context.TODO(), client.ObjectKey{Name: "test-cm", Namespace: ns}, obj); err != nil {
		return err
	}
	if err := cl.Delete(context.TODO(), obj); err != nil {
		return err
	}

	newobj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
    `)
	return cl.Create(context.TODO(), newobj, client.FieldOwner("alice"))
}

func slide9(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
    key: "different value" 
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("alice"))
}

func slide10(cl client.Client, ns string) error {
	obj := object(ns, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
    another-key: value 
    `)
	return cl.Patch(context.TODO(), obj, client.Apply, client.FieldOwner("alice"))
}

func exitIfError(err error) {
	if err != nil {
		panic(fmt.Errorf("error (%T): %w", err, err))
	}
}

func getClient() client.Client {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := rules.Load()
	exitIfError(err)

	restCfg, err := clientcmd.NewDefaultClientConfig(*cfg, nil).ClientConfig()
	exitIfError(err)

	scheme := runtime.NewScheme()
	exitIfError(corev1.AddToScheme(scheme))

	discoverCl, err := discovery.NewDiscoveryClientForConfig(restCfg)
	exitIfError(err)

	grs, err := restmapper.GetAPIGroupResources(discoverCl)
	exitIfError(err)

	cl, err := client.New(restCfg, client.Options{
		Scheme: scheme,
		Mapper: restmapper.NewDiscoveryRESTMapper(grs),
	})
	exitIfError(err)

	return cl
}

func deleteIfExist(cl client.Client, objs ...client.Object) {
	for _, obj := range objs {
		obj = obj.DeepCopyObject().(client.Object)
		if err := cl.Get(context.TODO(), client.ObjectKeyFromObject(obj), obj); err != nil {
			if errors.IsNotFound(err) {
				return
			}
			exitIfError(err)
		}
		if err := cl.Delete(context.TODO(), obj); err != nil && !errors.IsNotFound(err) {
			exitIfError(err)
		}
		for {
			if err := cl.Get(context.TODO(), client.ObjectKeyFromObject(obj), obj); errors.IsNotFound(err) {
				break
			}
		}
	}
}

func object(ns, yamlData string) client.Object {
	mapData := map[string]any{}
	exitIfError(yaml.Unmarshal([]byte(yamlData), &mapData))

	jsonData, err := json.Marshal(mapData)
	exitIfError(err)

	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, jsonData)
	exitIfError(err)
	co := obj.(client.Object)
	co.SetNamespace(ns)
	return co
}
