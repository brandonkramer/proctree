package proctree

import (
	"context"
	"strings"
	"testing"
)

func TestValidateSpec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		spec    Spec
		wantErr string
	}{
		{name: "clean exec", spec: Spec{Path: "/bin/echo", Args: []string{"hi"}}, wantErr: ""},
		{name: "null path", spec: Spec{Path: "echo\x00bad"}, wantErr: "invalid path"},
		{name: "null arg", spec: Spec{Path: "/bin/echo", Args: []string{"ok\x00bad"}}, wantErr: "invalid argument"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSpec(&tc.spec)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestRunSpecValidation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		spec    *Spec
		wantErr string
	}{
		{name: "nil spec", spec: nil, wantErr: "spec is nil"},
		{name: "empty spec", spec: &Spec{}, wantErr: "requires Path or Shell"},
		{name: "invalid path", spec: &Spec{Path: "bad\x00path"}, wantErr: "invalid path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Run(context.Background(), tc.spec, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
