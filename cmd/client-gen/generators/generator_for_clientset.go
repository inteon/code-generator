/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package generators

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	clientgentypes "k8s.io/code-generator/cmd/client-gen/types"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
)

// genClientset generates a package for a clientset.
type genClientset struct {
	generator.DefaultGen
	groups             []clientgentypes.GroupVersions
	groupGoNames       map[clientgentypes.GroupVersion]string
	clientsetPackage   string
	outputPackage      string
	imports            namer.ImportTracker
	clientsetGenerated bool
}

var _ generator.Generator = &genClientset{}

func (g *genClientset) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

// We only want to call GenerateType() once.
func (g *genClientset) Filter(c *generator.Context, t *types.Type) bool {
	ret := !g.clientsetGenerated
	g.clientsetGenerated = true
	return ret
}

func (g *genClientset) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	for _, group := range g.groups {
		for _, version := range group.Versions {
			typedClientPath := filepath.Join(g.clientsetPackage, "typed", strings.ToLower(group.PackageName), strings.ToLower(version.NonEmpty()))
			groupAlias := strings.ToLower(g.groupGoNames[clientgentypes.GroupVersion{Group: group.Group, Version: version.Version}])
			imports = append(imports, fmt.Sprintf("%s%s \"%s\"", groupAlias, strings.ToLower(version.NonEmpty()), typedClientPath))
		}
	}
	return
}

func (g *genClientset) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	// TODO: We actually don't need any type information to generate the clientset,
	// perhaps we can adapt the go2ild framework to this kind of usage.
	sw := generator.NewSnippetWriter(w, c, "$", "$")

	allGroups := clientgentypes.ToGroupVersionInfo(g.groups, g.groupGoNames)
	m := map[string]interface{}{
		"allGroups":                            allGroups,
		"Config":                               c.Universe.Type(types.Name{Package: "k8s.io/client-go/rest", Name: "Config"}),
		"DefaultKubernetesUserAgent":           c.Universe.Function(types.Name{Package: "k8s.io/client-go/rest", Name: "DefaultKubernetesUserAgent"}),
		"RESTClientInterface":                  c.Universe.Type(types.Name{Package: "k8s.io/client-go/rest", Name: "Interface"}),
		"RESTHTTPClientFor":                    c.Universe.Function(types.Name{Package: "k8s.io/client-go/rest", Name: "HTTPClientFor"}),
		"DiscoveryInterface":                   c.Universe.Type(types.Name{Package: "k8s.io/client-go/discovery", Name: "DiscoveryInterface"}),
		"DiscoveryClient":                      c.Universe.Type(types.Name{Package: "k8s.io/client-go/discovery", Name: "DiscoveryClient"}),
		"NewDiscoveryClientForConfigAndClient": c.Universe.Function(types.Name{Package: "k8s.io/client-go/discovery", Name: "NewDiscoveryClientForConfigAndClient"}),
		"NewDiscoveryClientForConfigOrDie":     c.Universe.Function(types.Name{Package: "k8s.io/client-go/discovery", Name: "NewDiscoveryClientForConfigOrDie"}),
		"NewDiscoveryClient":                   c.Universe.Function(types.Name{Package: "k8s.io/client-go/discovery", Name: "NewDiscoveryClient"}),
		"flowcontrolNewTokenBucketRateLimiter": c.Universe.Function(types.Name{Package: "k8s.io/client-go/util/flowcontrol", Name: "NewTokenBucketRateLimiter"}),
		"restRESTClientForConfigAndClient":     c.Universe.Function(types.Name{Package: "k8s.io/client-go/rest", Name: "RESTClientForConfigAndClient"}),
	}
	sw.Do(clientsetInterface, m)
	sw.Do(clientsetTemplate, m)
	for _, g := range allGroups {
		sw.Do(clientsetInterfaceImplTemplate, g)
	}
	sw.Do(getDiscoveryTemplate, m)
	sw.Do(newClientsetForRESTClientTemplate, m)

	return sw.Error()
}

var clientsetInterface = `
type Interface interface {
	Discovery() $.DiscoveryInterface|raw$
    $range .allGroups$$.GroupGoName$$.Version$() $.PackageAlias$.$.GroupGoName$$.Version$Interface
	$end$
}
`

var clientsetTemplate = `
// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*$.DiscoveryClient|raw$
    client $.RESTClientInterface|raw$
}
`

var clientsetInterfaceImplTemplate = `
// $.GroupGoName$$.Version$ retrieves the $.GroupGoName$$.Version$Client
func (c *Clientset) $.GroupGoName$$.Version$() $.PackageAlias$.$.GroupGoName$$.Version$Interface {
	return $.PackageAlias$.New(c.client)
}
`

var getDiscoveryTemplate = `
// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() $.DiscoveryInterface|raw$ {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}
`

var newClientsetForRESTClientTemplate = `
// New creates a new Clientset for the given RESTClient.
func New(c $.RESTClientInterface|raw$) *Clientset {
	return &Clientset{
		client: c,
		DiscoveryClient: $.NewDiscoveryClient|raw$(c),
	}
}
`
