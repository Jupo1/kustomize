/*
Copyright 2018 The Kubernetes Authors.

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

package resource

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"sigs.k8s.io/kustomize/internal/kusterr"
	"sigs.k8s.io/kustomize/pkg/ifc"
	"sigs.k8s.io/kustomize/pkg/types"
)

// Factory makes instances of Resource.
type Factory struct {
	kf ifc.KunstructuredFactory
}

// NewFactory makes an instance of Factory.
func NewFactory(kf ifc.KunstructuredFactory) *Factory {
	return &Factory{kf: kf}
}

// FromMap returns a new instance of Resource.
func (rf *Factory) FromMap(m map[string]interface{}) *Resource {
	return &Resource{
		Kunstructured: rf.kf.FromMap(m),
		options:       types.NewGenArgs(nil, nil),
	}
}

// FromMapAndOption returns a new instance of Resource with given options.
func (rf *Factory) FromMapAndOption(m map[string]interface{}, args *types.GeneratorArgs, option *types.GeneratorOptions) *Resource {
	return &Resource{
		Kunstructured: rf.kf.FromMap(m),
		options:       types.NewGenArgs(args, option),
	}
}

// FromKunstructured returns a new instance of Resource.
func (rf *Factory) FromKunstructured(
	u ifc.Kunstructured) *Resource {
	if u == nil {
		log.Fatal("unstruct ifc must not be null")
	}
	return &Resource{
		Kunstructured: u,
		options:       types.NewGenArgs(nil, nil),
	}
}

// SliceFromPatches returns a slice of resources given a patch path
// slice from a kustomization file.
func (rf *Factory) SliceFromPatches(
	ldr ifc.Loader, paths []types.PatchStrategicMerge) ([]*Resource, error) {
	var result []*Resource
	for _, path := range paths {
		content, err := ldr.Load(string(path))
		if err != nil {
			return nil, err
		}
		res, err := rf.SliceFromBytes(content)
		if err != nil {
			return nil, kusterr.Handler(err, string(path))
		}
		result = append(result, res...)
	}
	return result, nil
}

// FromBytes unmarshalls bytes into one Resource.
func (rf *Factory) FromBytes(in []byte) (*Resource, error) {
	result, err := rf.SliceFromBytes(in)
	if err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, fmt.Errorf(
			"expected 1 resource, found %d in %v", len(result), in)
	}
	return result[0], nil
}

// SliceFromBytes unmarshalls bytes into a Resource slice.
func (rf *Factory) SliceFromBytes(in []byte) ([]*Resource, error) {
	kunStructs, err := rf.kf.SliceFromBytes(in)
	if err != nil {
		return nil, err
	}
	var result []*Resource
	for len(kunStructs) > 0 {
		u := kunStructs[0]
		kunStructs = kunStructs[1:]
		if strings.HasSuffix(u.GetKind(), "List") {
			items := u.Map()["items"]
			itemsSlice, ok := items.([]interface{})
			if !ok {
				if items == nil {
					// an empty list
					continue
				}
				return nil, fmt.Errorf("items in List is type %T, expected array", items)
			}
			for _, item := range itemsSlice {
				itemJSON, err := json.Marshal(item)
				if err != nil {
					return nil, err
				}
				innerU, err := rf.kf.SliceFromBytes(itemJSON)
				if err != nil {
					return nil, err
				}
				// append innerU to kunStructs so nested Lists can be handled
				kunStructs = append(kunStructs, innerU...)
			}
		} else {
			result = append(result, rf.FromKunstructured(u))
		}
	}
	return result, nil
}

// MakeConfigMap makes an instance of Resource for ConfigMap
func (rf *Factory) MakeConfigMap(
	ldr ifc.Loader,
	options *types.GeneratorOptions,
	args *types.ConfigMapArgs) (*Resource, error) {
	u, err := rf.kf.MakeConfigMap(ldr, options, args)
	if err != nil {
		return nil, err
	}
	return &Resource{
		Kunstructured: u,
		options: types.NewGenArgs(
			&types.GeneratorArgs{Behavior: args.Behavior},
			options),
	}, nil
}

// MakeSecret makes an instance of Resource for Secret
func (rf *Factory) MakeSecret(
	ldr ifc.Loader,
	options *types.GeneratorOptions,
	args *types.SecretArgs) (*Resource, error) {
	u, err := rf.kf.MakeSecret(ldr, options, args)
	if err != nil {
		return nil, err
	}
	return &Resource{
		Kunstructured: u,
		options: types.NewGenArgs(
			&types.GeneratorArgs{Behavior: args.Behavior},
			options),
	}, nil
}
