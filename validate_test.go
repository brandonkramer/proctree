package proctree

import (
	"context"
	"strings"
	"testing"
)

func TestValidateSpecRejectsNullPath(t *testing.T) {
	err := validateSpec(&Spec{Path: "echo\x00bad"})
	if err == nil || !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateSpecRejectsNullArgument(t *testing.T) {
	err := validateSpec(&Spec{Path: "/bin/echo", Args: []string{"ok\x00bad"}})
	if err == nil || !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateSpecAllowsCleanExecSpec(t *testing.T) {
	if err := validateSpec(&Spec{Path: "/bin/echo", Args: []string{"hi"}}); err != nil {
		t.Fatal(err)
	}
}

func TestRunNilSpec(t *testing.T) {
	_, err := Run(context.Background(), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "spec is nil") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunEmptySpec(t *testing.T) {
	_, err := Run(context.Background(), &Spec{}, nil)
	if err == nil || !strings.Contains(err.Error(), "requires Path or Shell") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunRejectsInvalidPath(t *testing.T) {
	_, err := Run(context.Background(), &Spec{Path: "bad\x00path"}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("err=%v", err)
	}
}
