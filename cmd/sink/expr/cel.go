// Copyright KubeArchive Authors
// SPDX-License-Identifier: Apache-2.0

package expr

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	celext "github.com/google/cel-go/ext"
	"github.com/kubearchive/kubearchive/cmd/operator/api/v1alpha1"
	"github.com/kubearchive/kubearchive/pkg/files"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const globalKey = "_global"

type Filters struct {
	archive map[string]cel.Program
	delete  map[string]cel.Program
}

// Returns a Filters struct with an empty archive and delete slice
func EmptyFilters() *Filters {
	return &Filters{
		archive: make(map[string]cel.Program),
		delete:  make(map[string]cel.Program),
	}
}

// Creates a Filters struct from the path to a directory where a ConfigMap was mounted. If path is empty string or path
// does not exist, it returns a Filters struct with empty archive and delete slices. It will attempt to create all the
// cel programs that it can from the ConfigMap. Any errors that are encountered are wrapped together and returned. Even
// if this function returns an error, the Filters struct returned can still be used and will not be nil.
func NewFilters(path string) (*Filters, error) {
	var errList []error
	filters := &Filters{
		archive: make(map[string]cel.Program),
		delete:  make(map[string]cel.Program),
	}
	exists, err := files.PathExists(path)
	if err != nil {
		return filters, fmt.Errorf("cannot determine if ConfigMap is mounted at path %s: %s", path, err)
	}
	if path == "" || !exists {
		return filters, fmt.Errorf("cannot create Filters. ConfigMap is not mounted at path %s", path)
	}
	filterFiles, err := files.DirectoryFiles(path)
	if err != nil {
		return filters, fmt.Errorf(
			"cannot create Filters. Could not read files created from ConfigMap mounted at path %s: %s",
			path,
			err,
		)
	}
	for namespace, filePath := range filterFiles {
		resourcesBytes, err := os.ReadFile(filePath)
		if err != nil {
			errList = append(errList, err)
			continue
		}
		resources := []v1alpha1.KubeArchiveConfigResource{}
		err = yaml.Unmarshal(resourcesBytes, &resources)
		if err != nil {
			errList = append(errList, err)
			continue
		}
		for _, resource := range resources {
			namespaceGvk := NamespaceGroupVersionKind(namespace, resource)
			archiveExpr, err := CreateCelExpr(resource.ArchiveWhen)
			if err != nil {
				errList = append(errList, err)
			} else {
				filters.archive[namespaceGvk] = *archiveExpr
			}
			deleteExpr, err := CreateCelExpr(resource.DeleteWhen)
			if err != nil {
				errList = append(errList, err)
			} else {
				filters.delete[namespaceGvk] = *deleteExpr
			}
		}
	}

	return filters, errors.Join(errList...)

}

// Returns whether obj needs to be archived. Obj needs to be archived if any of the cel programs in archive return true
// or if obj needs to be deleted. If obj is nil, it returns false
func (f *Filters) MustArchive(ctx context.Context, obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	mustArchive := f.MustDelete(ctx, obj)
	if mustArchive {
		return mustArchive
	}
	globalGvk := GlobalGroupVersionKind(obj)
	namespaceGvk := NamespaceGroupVersionKindFromObject(obj)
	program, exists := f.archive[globalGvk]
	if exists {
		mustArchive = executeCel(ctx, program, obj)
	}
	if mustArchive {
		return mustArchive
	}
	program, exists = f.archive[namespaceGvk]
	if exists {
		mustArchive = executeCel(ctx, program, obj)
	}
	return mustArchive
}

// Returns whether obj needs to be deleted. Obj needs to be deleted if the cel program in delete that matches the
// NamespaceGroupVersionKind of obj or the cel program for the GlobalGroupVersionKind in delete for obj returns true. If
// obj is nil, it returns false
func (f *Filters) MustDelete(ctx context.Context, obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	mustDelete := false
	globalGvk := GlobalGroupVersionKind(obj)
	namespaceGvk := NamespaceGroupVersionKindFromObject(obj)
	program, exists := f.delete[globalGvk]
	if exists {
		mustDelete = executeCel(ctx, program, obj)
	}
	if mustDelete {
		return mustDelete
	}
	program, exists = f.delete[namespaceGvk]
	if exists {
		mustDelete = executeCel(ctx, program, obj)
	}
	return mustDelete
}

// Returns a string that is the form <globalKey>:GroupVersionKind
func GlobalGroupVersionKind(obj *unstructured.Unstructured) string {
	if obj == nil {
		return ":"
	}
	return fmt.Sprintf("%s:%s", globalKey, obj.GetObjectKind().GroupVersionKind())
}

// Returns a string that is the form <namespace>:GroupVersionKind
func NamespaceGroupVersionKindFromObject(obj *unstructured.Unstructured) string {
	if obj == nil {
		return ":"
	}
	return fmt.Sprintf("%s:%s", obj.GetNamespace(), obj.GetObjectKind().GroupVersionKind())
}

// Returns a string that is the form <namespace>:GroupVersionKind
func NamespaceGroupVersionKind(namespace string, resource v1alpha1.KubeArchiveConfigResource) string {
	gvk := schema.FromAPIVersionAndKind(resource.Selector.APIVersion, resource.Selector.Kind)
	return fmt.Sprintf("%s:%s", namespace, gvk)
}

func CreateCelExpr(expr string) (*cel.Program, error) {
	mapStrDyn := decls.NewMapType(decls.String, decls.Dyn)
	env, err := cel.NewEnv(
		celext.Strings(),
		celext.Encoders(),
		celext.Sets(),
		celext.Lists(),
		celext.Math(),
		cel.Declarations(
			decls.NewVar("metadata", mapStrDyn),
			decls.NewVar("spec", mapStrDyn),
			decls.NewVar("status", mapStrDyn),
		),
	)
	if err != nil {
		return nil, err
	}
	parsed, issues := env.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	program, err := env.Program(checked)
	if err != nil {
		return nil, err
	}
	return &program, err
}

// executes the cel program with obj as the input. If the program returns and error or if the program returns a value
// that cannot be converted to bool, false is returned. Otherwise the value returned by the cel program is returned
func executeCel(ctx context.Context, program cel.Program, obj *unstructured.Unstructured) bool {
	val, _, err := program.ContextEval(ctx, obj.Object)
	if err != nil {
		return false
	}
	boolVal, ok := val.Value().(bool)
	if !ok {
		return false
	}
	return boolVal
}
