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

	// If it is an interface (which is very similar to a pointer), do basically the
	// same as for the pointer. Though a pointer is not the same as an interface so
	// note that we have to call Elem() after creating a new object because otherwise
	// we would end up with an actual pointer
	case reflect.Interface:
		// Get rid of the wrapping interface
		originalValue := original.Elem()
		// Create a new object. Now new gives us a pointer, but we want the value it
		// points to, so we have to call Elem() to unwrap it

		if originalValue.IsValid() {
			reflectType := originalValue.Type()

			reflectValue := reflect.New(reflectType)

			copyValue := reflectValue.Elem()
			if err := deeplyReplace(copyValue, originalValue, replaceMap); err != nil {
				// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
				return err
			}
			copy.Set(copyValue)
		}

	// If it is a struct we translate each field
	case reflect.Struct:
		for i := 0; i < original.NumField(); i++ {
			if err := deeplyReplace(copy.Field(i), original.Field(i), replaceMap); err != nil {
				// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
				return err
			}
		}

	// If it is a slice we create a new slice and translate each element
	case reflect.Slice:
		if copy.CanSet() {
			copy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		} else {
			copyValueIntoUnexported(copy, reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		}

		for i := 0; i < original.Len(); i++ {
			if err := deeplyReplace(copy.Index(i), original.Index(i), replaceMap); err != nil {
				// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
				return err
			}
		}

	// If it is a map we create a new map and translate each value
	case reflect.Map:
		if copy.CanSet() {
			copy.Set(reflect.MakeMap(original.Type()))
		} else {
			copyValueIntoUnexported(copy, reflect.MakeMap(original.Type()))
		}
		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			if originalValue.Kind() != reflect.String && isNillable(originalValue) && originalValue.IsNil() {
				continue
			}
			// New gives us a pointer, but again we want the value
			copyValue := reflect.New(originalValue.Type()).Elem()

			if err := deeplyReplace(copyValue, originalValue, replaceMap); err != nil {
				// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
				return err
			}

			// Keys can be templated as well as values (e.g. to template something into an annotation).
			if key.Kind() == reflect.String {
				templatedKey, err := replace(key.String(), replaceMap)
				if err != nil {
					// Not wrapping the error, since this is a recursive function. Avoids excessively long error messages.
					return err
				}
				key = reflect.ValueOf(templatedKey)
			}

			copy.SetMapIndex(key, copyValue)
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
		if copy.CanSet() {
			copy.Set(original)
		} else {
			copyUnexported(copy, original)
		}
	}
	return nil
}

// isNillable returns true if the value is something which may be set to nil. This function is meant to guard against a
// panic from calling IsNil on a non-pointer type.
func isNillable(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return true
	}
	return false
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
