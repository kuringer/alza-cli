package main

import (
	"errors"
	"testing"

	"github.com/kuringer/alza-cli/client"
)

type fakeCart struct {
	items     []client.CartItem
	removed   []int
	removeErr error
}

func (f *fakeCart) GetCart() ([]client.CartItem, error) { return f.items, nil }
func (f *fakeCart) RemoveFromCart(id int) error {
	f.removed = append(f.removed, id)
	return f.removeErr
}

func TestRemoveOrderedFromCartRemovesMatch(t *testing.T) {
	f := &fakeCart{items: []client.CartItem{{ProductID: 1}, {ProductID: 7}}}
	removed, err := removeOrderedFromCart(f, 7)
	if err != nil || !removed || len(f.removed) != 1 || f.removed[0] != 7 {
		t.Fatalf("removed=%v err=%v calls=%v", removed, err, f.removed)
	}
}

func TestRemoveOrderedFromCartNoMatch(t *testing.T) {
	f := &fakeCart{items: []client.CartItem{{ProductID: 1}}}
	removed, err := removeOrderedFromCart(f, 7)
	if err != nil || removed || len(f.removed) != 0 {
		t.Fatalf("removed=%v err=%v calls=%v", removed, err, f.removed)
	}
}

func TestRemoveOrderedFromCartError(t *testing.T) {
	f := &fakeCart{items: []client.CartItem{{ProductID: 7}}, removeErr: errors.New("boom")}
	removed, err := removeOrderedFromCart(f, 7)
	if err == nil || removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
}
