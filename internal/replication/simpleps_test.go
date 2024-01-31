package replication

import "testing"

func TestName(t *testing.T) {
	ps := NewSimplePrimaryStandby(&SimplePrimaryStandbyConfig{
		NodeName:       "",
		ClusterNodeMap: nil,
		RegHandler:     nil,
	})

	if err := ps.Initialize(); err != nil {
		t.Fatal(err)
	}
}
