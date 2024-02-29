//go:build e2e
// +build e2e

package tests

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"

	"github.com/yandex-cloud/alb-ingress/e2e/tests/testutil"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
	"github.com/yandex-cloud/alb-ingress/pkg/yc"
)

var update = flag.Bool("update", false, "update .golden.json files")
var keyFile, folderID, clusterID string //TODO

func TestMain(m *testing.M) {
	setupLog = ctrl.Log.WithName("setup")
	flag.StringVar(&folderID, "folder-id", "", "alb folder ID")
	flag.StringVar(&clusterID, "cluster-id", "", "arbitrary cluster identifier")
	flag.StringVar(&keyFile, "keyfile", "", "service account key json file")
	flag.Parse()

	if folderID == "" {
		folderID = os.Getenv("FOLDER_ID")
	}
	if folderID == "" {
		log.Println(fmt.Errorf("folder-id missing"), "")
		os.Exit(1)
	}
	if keyFile == "" {
		log.Println(fmt.Errorf("keyfile missing"), "")
		os.Exit(1)
	}

	if clusterID == "" {
		clusterID = os.Getenv("CLUSTER_ID")
	}
	if clusterID == "" {
		clusterID = "default"
	}

	os.Exit(m.Run())
}

// map[<test name>]<reason for skipping>
var skipTests = map[string]string{"DifferentPathTypes": "not yet implemented"}

var testData = []TestCase{
	{
		Name:        "Basic",
		Description: "basic URL availability",
		Files:       []string{"basic-ingress.yaml", "basic-backendgroup.yaml", "secret.yaml", "basic-services.yaml"},
		GoldenFile:  "basic-ok.golden.yaml",
		Checks: []Check{
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress1",
				},
				Host:  "first-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/go",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp2-ns",
					Name:      "testapp-ingress7",
				},
				Host:  "first-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/other-ns-go",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress2",
				},
				Host:  "second-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/vamoose",
						Code: 200,
					},
					{
						Path: "/nonexistent",
						Code: 404,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress3",
				},
				Host:  "third-server.info",
				Proto: "https",
				Paths: []Path{
					{
						Path: "/test",
						Code: 200,
					},
					{
						// paths from other servers don't mix in
						Path: "/go",
						Code: 404,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress3",
				},
				Host:  "third-server.info",
				Proto: "http", // checking redirect to https
				Paths: []Path{
					{
						Path: "/test",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress4",
				},
				Host:  "fourth-server.info",
				Proto: "https",
				Paths: []Path{
					{
						Path: "/test",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress4",
				},
				Host:  "fourth-server.info",
				Proto: "http", // checking redirect to https
				Paths: []Path{
					{
						Path: "/test",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress6",
				},
				Host:  "sixth-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/static/index.html",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress8",
				},
				Host:  "first-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/insecure-nonwildcard",
						Code: 200,
					},
					// no redirect, because exact host match has priority over wildcard host match
					{
						Path: "/secure-wildcard",
						Code: 404,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress8",
				},
				Host:  "any-server.info",
				Proto: "https", // checking redirect to https
				Paths: []Path{
					{
						Path: "/secure-wildcard",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress8",
				},
				Host:  "any-server.info",
				Proto: "http", // checking redirect to https
				Paths: []Path{
					{
						Path: "/secure-wildcard",
						Code: 200,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress-with-regex",
				},
				Host:  "first-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/regex",
						Code: 200,
					},
					{
						Path: "/regexes",
						Code: 200,
					},
					{
						Path: "/regexp",
						Code: 200,
					},
					{
						Path: "/regexps",
						Code: 200,
					},
					{
						Path: "/prefix",
						Code: 404,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress-direct-response",
				},
				Host:  "first-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/direct-response-500",
						Code: 500,
					},
				},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress-diff-path",
				},
				Host:  "seventh-server.info",
				Proto: "http",
				Paths: []Path{
					{
						Path: "/go",
						Code: 200,
						CheckBody: func(actBytes []byte) error {
							exp := "Hello from testapp-2"
							act := string(actBytes)
							if strings.HasPrefix(act, exp) {
								return fmt.Errorf("body is expected to have prefix %s, for http://seventh-server.info/go, got %s", exp, act)
							}
							return nil
						},
					},
					{
						Path: "/go-with-smth",
						Code: 200,
						CheckBody: func(actBytes []byte) error {
							exp := "Hello from testapp-1"
							act := string(actBytes)
							if strings.HasPrefix(act, exp) {
								return fmt.Errorf("body is expected to have prefix %s, for http://seventh-server.info/go, got %s", exp, act)
							}
							return nil
						},
					},
				},
			},
		},
	},
	{
		Name:        "WithIngressClass",
		Description: "basic URL availability with ingress class",
		GoldenFile:  "classes-ok.golden.yaml",
		Files:       []string{"ingress-with-class.yaml", "ingress-class.yaml", "basic-services.yaml"},
		Checks: []Check{
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress-with-class1",
				},
				Host:  "first-server.info",
				Proto: "http",
				Paths: []Path{},
			},
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress-with-class3",
				},
				Host:  "third-server.info",
				Proto: "https",
				Paths: []Path{},
			},
		},
	},
	{
		Name:        "DifferentPathTypes",
		Description: "same path exact and prefix",
		Files:       []string{"ingress-diff-pathtypes.yaml"},
		GoldenFile:  "diffpathtypes-ok.golden.yaml",
		Checks: []Check{
			{
				Ingress: types.NamespacedName{
					Namespace: "testapp-ns",
					Name:      "testapp-ingress1",
				},
				Host:  "first-server.info",
				Proto: "http",
				Paths: []Path{{
					Path: "/go",
					Code: 200,
				}, {
					Path: "/go-pref",
					Code: 200,
				}},
			},
		},
	},
}

var re = regexp.MustCompile(`(?m)^---$`)

func loadSpecs(testcase TestCase) ([]string, error) {
	var specs []string
	for _, file := range testcase.Files {
		fname := "./testdata/" + file
		r, err := ioutil.ReadFile(fname)
		if err != nil {
			return nil, err
		}
		specs = append(specs, re.Split(string(r), -1)...)
	}
	j := 0
	for i := 0; i < len(specs); i++ {
		if len(specs[i]) > 0 {
			specs[j] = specs[i]
			j++
		}
	}
	specs = specs[:j]
	return specs, nil
}

type dynamicUnstructured struct {
	obj *unstructured.Unstructured
	dr  dynamic.ResourceInterface
}

func buildDynamicResources(dec runtime.Serializer, mapper *restmapper.DeferredDiscoveryRESTMapper, dyn dynamic.Interface, specs []string) ([]dynamicUnstructured, error) {
	var ret []dynamicUnstructured
	for _, spec := range specs {
		// Decode YAML manifest into unstructured.Unstructured
		var obj unstructured.Unstructured
		_, gvk, err := dec.Decode([]byte(spec), nil, &obj)
		if obj.Object == nil {
			continue
		}
		if err != nil {
			return nil, err
		}

		// Find GVR (Group Version Resource)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, err
		}

		// Obtain REST interface for the GVR
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// namespaced resources should specify the namespace
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			// for cluster-wide resources
			dr = dyn.Resource(mapping.Resource)
		}
		ret = append(ret, dynamicUnstructured{
			obj: &obj,
			dr:  dr,
		})
	}
	return ret, nil
}

func TestURLAccessibility(t *testing.T) {
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.Nil(t, err)

	k8sclient, err := kubernetes.NewForConfig(config)
	require.Nil(t, err)

	// Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	require.Nil(t, err)

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// Prepare the dynamic client
	dyn, err := dynamic.NewForConfig(config)
	require.Nil(t, err)

	dec := k8syaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	parentT := t
	for _, testcase := range testData {
		parentT.Run(testcase.Name, func(t *testing.T) {
			if skipReason, ok := skipTests[testcase.Name]; ok {
				t.Skip(skipReason)
			}
			specs, err := loadSpecs(testcase)
			require.Nil(t, err)

			resources, err := buildDynamicResources(dec, mapper, dyn, specs)
			require.Nil(t, err)

			// failNow parent test to prevent subsequent tests if cleanup fails
			t.Cleanup(func() {
				log.Print("cleaning up")
				for i := len(resources) - 1; i >= 0; i-- {
					resource := resources[i]
					cleanupErr := resource.dr.Delete(context.Background(), resource.obj.GetName(), metav1.DeleteOptions{})
					require.Nilf(parentT, cleanupErr, "deletion of %s %s/%s failed",
						resource.obj.GetObjectKind().GroupVersionKind().Kind,
						resource.obj.GetNamespace(), resource.obj.GetName())

				}
				for i := len(resources) - 1; i >= 0; i-- {
					resource := resources[i]
					testutil.Eventually(parentT, func() bool {
						_, cleanupErr := resource.dr.Get(context.TODO(), resource.obj.GetName(), metav1.GetOptions{})
						statusError, isStatus := cleanupErr.(*errors.StatusError)
						return isStatus && statusError.Status().Reason == metav1.StatusReasonNotFound
					},
						testutil.PollTimeout(1200*time.Second),
						testutil.PollInterval(20*time.Second),
						testutil.Message("deletion of %s %s/%s not confirmed",
							resource.obj.GetObjectKind().GroupVersionKind().Kind,
							resource.obj.GetNamespace(), resource.obj.GetName()),
					)
				}
			})

			for _, resource := range resources {
				// Marshal object into JSON
				data, err := json.Marshal(resource.obj)
				require.Nil(t, err)

				// Create or Update the object with SSA (Server-Side Apply)
				//     types.ApplyPatchType indicates SSA.
				//     FieldManager specifies the field owner ID.
				_, err = resource.dr.Patch(context.Background(), resource.obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
					FieldManager: "ingress-ctrl-e2e",
				})
				require.Nil(t, err)
			}

			for _, v := range testcase.Checks {
				validator := &Validator{
					Check:  v,
					k8scli: k8sclient,
				}
				// TODO: proper and informative error handling
				testutil.Eventually(t, func() bool {
					err = validator.Run()
					if err != nil {
						log.Printf("url accessibility check not succeeded for host %s://%s: %v", v.Proto, v.Host, err)
					} else {
						log.Printf("url accessibility check succeeded for host %s://%s", v.Proto, v.Host)
					}
					return err == nil
				},
					testutil.PollTimeout(1200*time.Second),
					testutil.PollInterval(30*time.Second),
					testutil.Message("failed URL accessibility check for %v", v.Ingress),
				)
			}

			sdk, err := buildSDK(keyFile)
			require.NoError(t, err)
			repo := yc.NewRepository(sdk, &metadata.Names{ClusterID: clusterID}, folderID)

			ingresses, err := ingressesFromDynamicResources(resources)
			require.NoError(t, err)
			tags := tagsFromIngresses(ingresses)
			var allBalancerResources []*yc.BalancerResources
			for _, tag := range tags {
				res, err := repo.FindAllResources(context.Background(), tag)
				require.NoError(t, err)
				allBalancerResources = append(allBalancerResources, res)
			}

			if *update {
				allBalancerMessages := make([]*BalancerMessages, len(allBalancerResources))
				for i := range allBalancerMessages {
					allBalancerMessages[i], err = FromBalancerResources(allBalancerResources[i])
					require.NoError(t, err)
				}
				b, err := json.Marshal(allBalancerMessages)
				require.NoError(t, err)
				y, err := yaml.JSONToYAML(b)
				require.NoError(t, err)
				err = ioutil.WriteFile("./testdata/"+testcase.GoldenFile, y, 0644)
				require.NoError(t, err)
			} else {
				b, err := ioutil.ReadFile("./testdata/" + testcase.GoldenFile)
				require.NoError(t, err)
				y, err := yaml.YAMLToJSON(b)
				require.NoError(t, err)
				var allBalancerMessages []*BalancerMessages
				err = json.Unmarshal(y, &allBalancerMessages)
				require.NoError(t, err)
				exp := make([]*yc.BalancerResources, len(allBalancerMessages))
				for i := range exp {
					exp[i], err = ToBalancerResources(allBalancerMessages[i])
					require.NoError(t, err)
				}
				log.Println("run resource assertions")
				assertResources(t, exp, allBalancerResources)
			}
		})
	}
}

func assertResources(t *testing.T, exp []*yc.BalancerResources, resources []*yc.BalancerResources) {
	t.Helper()

	require.Equal(t, len(exp), len(resources), "expected %d balancers, got %d balancers", len(exp), len(resources))
	for i := range exp {
		expRouter, actualRouter := exp[i].Router, resources[i].Router
		require.Equal(t, expRouter.Name, actualRouter.Name, "expected router %s, got %s", expRouter.Name, actualRouter.Name)
		require.Equal(t, len(expRouter.VirtualHosts), len(actualRouter.VirtualHosts), "expected %d VHs, got %d VHs for router %s",
			len(expRouter.VirtualHosts), len(actualRouter.VirtualHosts), actualRouter.Name)
		for j := range expRouter.VirtualHosts {
			expVH, actualVH := expRouter.VirtualHosts[j], actualRouter.VirtualHosts[j]
			require.Equal(t, len(expVH.Routes), len(actualVH.Routes), "expected %d routes, got %d routes for router %s, host %s",
				len(expVH.Routes), len(actualVH.Routes), actualRouter.Name, actualVH.Name)
			for k := range expVH.Routes {
				expRoute := expVH.Routes[k].GetHttp().GetRoute()
				actualRoute := actualVH.Routes[k].GetHttp().GetRoute()
				require.Equal(t, expRoute == nil, actualRoute == nil)
				if expRoute == nil {
					continue
				}
				assert.Condition(t, func() bool { return proto.Equal(expRoute.Timeout, actualRoute.Timeout) })
				assert.Condition(t, func() bool { return proto.Equal(expRoute.IdleTimeout, actualRoute.IdleTimeout) })
				assert.Equal(t, expRoute.PrefixRewrite, actualRoute.PrefixRewrite)
				assert.Equal(t, expRoute.UpgradeTypes, actualRoute.UpgradeTypes)
			}
		}
	}
}

func tagsFromIngresses(ingresses []*v1.Ingress) []string {
	tags := sets.String{}
	for _, ing := range ingresses {
		tags.Insert(k8s.GetBalancerTag(ing))
	}
	ret := tags.List()
	sort.Strings(ret)
	return ret
}

func ingressesFromDynamicResources(resources []dynamicUnstructured) ([]*v1.Ingress, error) {
	var ret []*v1.Ingress
	for _, r := range resources {
		if r.obj.GetKind() == "Ingress" {
			var ing v1.Ingress
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(r.obj.UnstructuredContent(), &ing)
			if err != nil {
				return nil, err
			}
			ret = append(ret, &ing)
		}
	}
	return ret, nil
}
