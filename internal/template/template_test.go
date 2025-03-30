package template

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type unsupported struct {
	field []string
}

func Test_deeplyReplace(t *testing.T) {
	t.Run("unsupported field type", func(t *testing.T) {
		u := unsupported{field: []string{"a", "b"}}
		o := reflect.ValueOf(u)
		c := reflect.New(o.Type()).Elem()
		err := deeplyReplace(c, o, map[string]any{"a": "b"})
		require.Error(t, err)
	})
}
