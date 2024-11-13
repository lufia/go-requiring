package validator

import (
	"testing"
)

func TestMapDefault(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v := Map[map[string]string](
			For("key", Required[string]()),
			ForAny(Required[string]()),
		)
		testValidate(t, v, map[string]string{
			"key": "ok",
		}, "")
		testValidate(t, v, map[string]string(nil), "")
	})
}
