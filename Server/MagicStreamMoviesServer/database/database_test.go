package database

import "testing"

func TestOpenCollectionPanicsWithoutClient(t *testing.T) {
	oldClient := Client
	Client = nil
	t.Cleanup(func() { Client = oldClient })
	defer func() {
		if recover() == nil {
			t.Fatal("OpenCollection() did not panic for an uninitialized client")
		}
	}()
	OpenCollection("users")
}
