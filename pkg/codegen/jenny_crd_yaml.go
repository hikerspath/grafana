package codegen

import (
	"fmt"
	"path/filepath"

	cueopenapi "cuelang.org/go/encoding/openapi"
	cueyaml "cuelang.org/go/pkg/encoding/yaml"
	"github.com/grafana/codejen"
	"github.com/grafana/grafana/pkg/kindsys"
	"github.com/grafana/grafana/pkg/kindsys/k8ssys"
	"github.com/grafana/thema"
	"github.com/grafana/thema/encoding/openapi"
	goyaml "gopkg.in/yaml.v3"
)

// YamlCRDJenny generates a representation of a core structured kind in YAML CRD form.
func YamlCRDJenny(path string) OneToOne {
	return yamlCRDJenny{
		parentpath: path,
	}
}

type yamlCRDJenny struct {
	parentpath string
}

func (yamlCRDJenny) JennyName() string {
	return "YamlCRDJenny"
}

func (j yamlCRDJenny) Generate(decl *DeclForGen) (*codejen.File, error) {
	if !decl.IsCoreStructured() {
		return nil, nil
	}

	props := decl.SomeDecl.Properties.(kindsys.CoreStructuredProperties)
	lin := decl.Lineage()

	// We need to go through every schema, as they all have to be defined in the CRD
	sch, err := lin.Schema(thema.SV(0, 0))
	if err != nil {
		return nil, err
	}

	// FIXME no hardcode yo
	scope := "cluster"

	resource := customResourceDefinition{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Metadata: customResourceDefinitionMetadata{
			Name: fmt.Sprintf("%s.%s", props.PluralMachineName, props.Group),
		},
		Spec: k8ssys.CustomResourceDefinitionSpec{
			Group: props.Group,
			Scope: scope,
			Names: k8ssys.CustomResourceDefinitionSpecNames{
				Kind:   props.Name,
				Plural: props.PluralMachineName,
			},
			Versions: make([]k8ssys.CustomResourceDefinitionSpecVersion, 0),
		},
	}
	latest := thema.LatestVersion(lin)

	for sch != nil {
		ver, err := valueToCRDSpecVersion(sch, versionString(sch.Version()), sch.Version() == latest)
		if err != nil {
			return nil, err
		}
		resource.Spec.Versions = append(resource.Spec.Versions, ver)
		sch = sch.Successor()
	}
	contents, err := goyaml.Marshal(resource)
	if err != nil {
		return nil, err
	}

	return &codejen.File{
		RelativePath: filepath.Join(j.parentpath, props.MachineName, "crd", props.MachineName+".crd.yml"),
		Data:         contents,
	}, nil
}

// customResourceDefinition differs from k8ssys.CustomResourceDefinition in that it doesn't use the metav1
// TypeMeta and ObjectMeta, as those do not contain YAML tags and get improperly serialized to YAML.
// Since we don't need to use it with the kubernetes go-client, we don't need the extra functionality attached.
//
//nolint:lll
type customResourceDefinition struct {
	Kind       string                              `json:"kind,omitempty" yaml:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	APIVersion string                              `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty" protobuf:"bytes,2,opt,name=apiVersion"`
	Metadata   customResourceDefinitionMetadata    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec       k8ssys.CustomResourceDefinitionSpec `json:"spec"`
}

type customResourceDefinitionMetadata struct {
	Name string `json:"name,omitempty" yaml:"name" protobuf:"bytes,1,opt,name=name"`
	// TODO: other fields as necessary for codegen
}

type cueOpenAPIEncoded struct {
	Components cueOpenAPIEncodedComponents `json:"components"`
}

type cueOpenAPIEncodedComponents struct {
	Schemas map[string]any `json:"schemas"`
}

func valueToCRDSpecVersion(sch thema.Schema, name string, stored bool) (k8ssys.CustomResourceDefinitionSpecVersion, error) {
	f, err := openapi.GenerateSchema(sch, &cueopenapi.Config{
		ExpandReferences: true,
	})
	if err != nil {
		return k8ssys.CustomResourceDefinitionSpecVersion{}, err
	}

	str, err := cueyaml.Marshal(sch.Lineage().Runtime().Context().BuildFile(f))
	if err != nil {
		return k8ssys.CustomResourceDefinitionSpecVersion{}, err
	}

	// Decode the bytes back into an object where we can trim the openAPI clutter out
	// and grab just the schema as a map[string]any (which is what k8s wants)
	back := cueOpenAPIEncoded{}
	err = goyaml.Unmarshal([]byte(str), &back)
	if err != nil {
		return k8ssys.CustomResourceDefinitionSpecVersion{}, err
	}
	if len(back.Components.Schemas) != 1 {
		fmt.Println(len(back.Components.Schemas))
		// fmt.Println(back.Components.Schemas)
		// There should only be one schema here...
		// TODO: this may change with subresources--but subresources should have defined names
		return k8ssys.CustomResourceDefinitionSpecVersion{}, fmt.Errorf("version %s has multiple schemas", name)
	}
	var def map[string]any
	for _, v := range back.Components.Schemas {
		ok := false
		def, ok = v.(map[string]any)
		if !ok {
			return k8ssys.CustomResourceDefinitionSpecVersion{},
				fmt.Errorf("error generating openapi schema - generated schema has invalid type")
		}
	}

	return k8ssys.CustomResourceDefinitionSpecVersion{
		Name:    name,
		Served:  true,
		Storage: stored,
		Schema: map[string]any{
			"openAPIV3Schema": map[string]any{
				"properties": map[string]any{
					"spec": def,
				},
				"required": []any{
					"spec",
				},
				"type": "object",
			},
		},
	}, nil
}

func versionString(version thema.SyntacticVersion) string {
	return fmt.Sprintf("v%d-%d", version[0], version[1])
}
