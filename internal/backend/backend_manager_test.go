package backend

import (
	"go-balancer/internal/balancer/config"
	"testing"
)

func TestBackendManagerAddBackends(t *testing.T) {
	bm := NewBackendManager([]config.BackendInfo{
		config.NewBackendInfo("abc", 80),
	})

	newUrls := []config.BackendInfo{
		config.NewBackendInfo("def", 80),
	}

	bm.AddBackends(newUrls)

	l := bm.GetBackendCount()
	if l != 2 {
		t.Errorf("Failed length check after add: got %d expected 2", l)
	} else {
		u := bm.GetBackend(1).url
		if *u != *newUrls[0].URL {
			t.Errorf("Failed get backend after add: got %s expected %s", u, newUrls[0].URL)
		}
	}
}

func TestBackendManagerRemoveBackends(t *testing.T) {
	startUrl := []config.BackendInfo{
		config.NewBackendInfo("abc", 80),
		config.NewBackendInfo("def", 80),
		config.NewBackendInfo("ghi", 80),
		config.NewBackendInfo("jkl", 80),
	}
	bm := NewBackendManager(startUrl)

	removeUrl := []config.BackendInfo{
		config.NewBackendInfo("def", 80),
		config.NewBackendInfo("jkl", 80),
	}

	bm.RemoveBackends(removeUrl)

	l := bm.GetBackendCount()
	if l != 2 {
		t.Errorf("Failed length check after remove: got %d expected 2", l)
	} else {
		u1 := bm.GetBackend(0).url
		if *u1 != *startUrl[0].URL {
			t.Errorf("Failed get backend after remove: got %s expected %s", u1, startUrl[0].URL)
		}

		u2 := bm.GetBackend(1).url
		if *u2 != *startUrl[2].URL {
			t.Errorf("Failed get backend after remove: got %s expected %s", u2, startUrl[2].URL)
		}
	}
}
