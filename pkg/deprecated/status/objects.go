/*
Copyright 2018 The Tekton Authors

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

package status

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

// Rollout contains one or more Objects to rollout
type Rollout struct {
	Icon    string
	Status  string
	Path    string
	Objects []*Object
}

type Rollouts struct {
	Name     string
	Status   string
	Icon     string
	Rollouts []*Rollout
}

// Object encapsulates the metadata and state for an object
type Object struct {
	parsed *unstructured.Unstructured
	Raw    []byte
	JSON   []byte
	runtime.Object
	schema.GroupVersionKind
	types.NamespacedName

	ApplyStatus          string
	RolloutStatus        string
	RolloutStatusHistory []string
	Done                 bool
}

// Display returns the display name of a object
func (o *Object) Display() string {
	if o.Group != "" {
		return fmt.Sprintf("%s.%s.%s \"%s/%s\"",
			strings.ToLower(o.Kind), o.Group, o.Version, o.Namespace, o.Name)
	}
	return fmt.Sprintf("%s.%s \"%s/%s\"",
		strings.ToLower(o.Kind), o.Version, o.Namespace, o.Name)
}

// ParseObject parses json or yaml config processed by kustomize into an object
func ParseObject(o []byte) (*Object, error) {
	// Parse the unstructured data
	var err error
	obj := &Object{Raw: o}
	if obj.parsed, obj.JSON, err = parseUnstruct(o); err != nil {
		return nil, err
	}

	// Set NamespacedName
	meta, ok := obj.parsed.Object["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parseObject metadata is not a map[string]interface{}: %T", obj.parsed.Object["metadata"])
	}
	obj.Name = fmt.Sprintf("%s", meta["name"])
	if meta["namespace"] != nil && meta["namespace"] != "" {
		obj.Namespace = fmt.Sprintf("%s", meta["namespace"])
	} else {
		// Set a default namespacef it is empty
		obj.Namespace = "default"
	}

	// Set GroupVersionKind
	obj.Kind = fmt.Sprintf("%s", obj.parsed.Object["kind"])
	parts := strings.Split(fmt.Sprintf("%s", obj.parsed.Object["apiVersion"]), "/")
	if len(parts) == 1 {
		obj.Group = ""
		obj.Version = parts[0]
	} else if len(parts) == 2 {
		obj.Group = parts[0]
		obj.Version = parts[1]
	} else {
		return nil, fmt.Errorf("apiVersion not recognized %v", obj.parsed.Object["apiVersion"])
	}

	if obj.Object, err = scheme.Scheme.New(obj.GroupVersionKind); err != nil {
		// Object type not registered with the scheme.  May be +versioned skewed or an extension.
		// Use the unstructured object as the runtime.Object
		obj.Object = obj.parsed
		return obj, nil
	}

	// Object found in scheme.  Use the go-struct as the runtime.Object
	if err := json.Unmarshal(obj.JSON, obj.Object); err != nil {
		return nil, fmt.Errorf("could not unmarshal json %v\n%s", err, obj.Raw)
	}
	return obj, err
}

// parseUnstruct parses text into an Unstructured object
func parseUnstruct(o []byte) (*unstructured.Unstructured, []byte, error) {
	y := map[string]interface{}{}
	if err := yaml.Unmarshal(o, &y); err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal yaml %v\n%s", err, o)
	}

	o, err := json.Marshal(y)
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal json %v", err)
	}

	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(o); err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal json %v\n%s", err, o)
	}

	return u, o, nil
}
