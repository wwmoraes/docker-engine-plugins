package common

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	sm "github.com/cch123/supermonkey"
)

func TestAssert(t *testing.T) {
	patch := sm.Patch(os.Exit, func(code int) {
		panic(fmt.Sprintf("os.Exit(%d)", code))
	})
	defer patch.Unpatch()

	want := "os.Exit(1)"

	t.Run("exit 1 on error", func(t *testing.T) {
		defer func() {
			err := recover()
			if err == nil {
				t.Fatal("error expected, got nil")
			}

			got, ok := err.(string)
			if !ok {
				t.Fatalf("expected a string panic value, got %s", reflect.TypeOf(err))
			}
			if want != got {
				t.Fatalf("want %s, got %s", want, got)
			}
		}()

		Assert(fmt.Errorf("common: test"))
	})

	t.Run("ignore without error", func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				t.Fatalf("no error expected, got %#+v", err)
			}
		}()

		Assert(nil)
	})
}
