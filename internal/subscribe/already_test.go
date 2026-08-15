package subscribe

import (
	"errors"
	"testing"
)

func TestIsAlreadyListening(t *testing.T) {
	if !isAlreadyListening(errors.New("小红书房间 1 已在监听中")) {
		t.Fatal("zh")
	}
	if !isAlreadyListening(errors.New("service already exists")) {
		t.Fatal("en")
	}
	if isAlreadyListening(errors.New("connection refused")) {
		t.Fatal("other")
	}
	if isAlreadyListening(nil) {
		t.Fatal("nil")
	}
}
