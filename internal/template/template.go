package template

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"text/template"
	"unsafe"

	"github.com/crenshaw-dev/cluster-set/api/v1alpha1"
	sprig "github.com/go-task/slim-sprig/v3"
)

// This is a simplified version of the templating code in Argo CD's ApplicationSet.

var sprigFuncMap = sprig.GenericFuncMap() // a singleton for better performance

func init() {
	// Avoid allowing the user to learn things about the environment.
	delete(sprigFuncMap, "env")
	delete(sprigFuncMap, "expandenv")
	delete(sprigFuncMap, "getHostByName")
}

func Render(tmpl *v1alpha1.ClusterTemplate, params map[string]any) (*v1alpha1.ClusterTemplate, error) {
	if tmpl == nil {
		return nil, errors.New("cluster template is empty")
	}

	if len(params) == 0 {
		return tmpl, nil
	}

	original := reflect.ValueOf(tmpl)
	c := reflect.New(original.Type()).Elem()

	if err := deeplyReplace(c, original, params); err != nil {
		return nil, err
	}

	replacedTmpl := c.Interface().(*v1alpha1.ClusterTemplate)

	return replacedTmpl, nil
}

func copyValueIntoUnexported(destination, value reflect.Value) {
	reflect.NewAt(destination.Type(), unsafe.Pointer(destination.UnsafeAddr())).
		Elem().
		Set(value)
}

func copyUnexported(copy, original reflect.Value) {
	unexported := reflect.NewAt(original.Type(), unsafe.Pointer(original.UnsafeAddr())).Elem()
	copyValueIntoUnexported(copy, unexported)
}

// This function is in charge of searching all String fields of the object recursively and apply templating
// thanks to https://gist.github.com/randallmlough/1fd78ec8a1034916ca52281e3b886dc7
func deeplyReplace(copy, original reflect.Value, replaceMap map[string]any) error {
	switch original.Kind() {
	// The first cases handle nested structures and translate them recursively
	// If it is a pointer we need to unwrap and call once again
	case reflect.Ptr:
		// To get the actual value of the original we have to call Elem()
		// At the same time this unwraps the pointer so we don't end up in
		// an infinite recursion
		originalValue := original.Elem()
		// Check if the pointer is nil
		if !originalValue.IsValid() {
			return nil
		}
		// Allocate a new object and set the pointer to it
		if originalValue.CanSet() {
			copy.Set(reflect.New(originalValue.Type()))
		} else {
			copyUnexported(copy, original)
		}
		// Unwrap the newly created pointer
		if err := deeplyReplace(copy.Elem(), originalValue, replaceMap); err != nil {
			// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
			return err
		}

	// If it is a struct we translate each field
	case reflect.Struct:
		for i := 0; i < original.NumField(); i++ {
			if err := deeplyReplace(copy.Field(i), original.Field(i), replaceMap); err != nil {
				// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
				return err
			}
		}

	// Otherwise we cannot traverse anywhere so this finishes the recursion
	// If it is a string translate it (yay finally we're doing what we came for)
	case reflect.String:
		strToTemplate := original.String()
		templated, err := replace(strToTemplate, replaceMap)
		if err != nil {
			// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
			return err
		}
		if copy.CanSet() {
			copy.SetString(templated)
		} else {
			copyValueIntoUnexported(copy, reflect.ValueOf(templated))
		}
		return nil

	// And everything else will simply be taken from the original
	default:
		// We only support pointers, structs, or strings. More complex types should be represented as YAML/JSON strings
		// for the greatest templating flexibility.
		return fmt.Errorf("failed to template field of type %T: must be pointer, struct, or string", original)
	}
	return nil
}

// Replace executes basic string substitution of a template with replacement values.
// remaining in the substituted template.
func replace(tmpl string, replaceMap map[string]any) (string, error) {
	t, err := template.New("").Funcs(sprigFuncMap).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", tmpl, err)
	}

	var replacedTmplBuffer bytes.Buffer
	if err = t.Execute(&replacedTmplBuffer, replaceMap); err != nil {
		return "", fmt.Errorf("failed to execute go template %s: %w", tmpl, err)
	}

	return replacedTmplBuffer.String(), nil
}
