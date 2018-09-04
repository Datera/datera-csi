package client

import (
	"reflect"
	"testing"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	udc "github.com/Datera/go-udc/pkg/udc"
)

const WIM = 500

func createVolume(t *testing.T, client *DateraClient, v *VolOpts) (string, *Volume, func()) {
	name := "my-test-vol-" + dsdk.RandString(5)
	vol, err := client.CreateVolume(name, v, true)
	if err != nil {
		t.Fatal(err)
	}
	return name, vol, func() {
		if err = client.DeleteVolume(name, true); err != nil {
			t.Fatal(err)
		}
	}

}

func getClient(t *testing.T) *DateraClient {
	conf, err := udc.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewDateraClient(conf)
	if err != nil {
		t.Fatal(err)
	}
	client.NewContext(nil)
	return client
}

func TestVolumeCreate(t *testing.T) {
	client := getClient(t)
	v := &VolOpts{
		Size:         5,
		Replica:      1,
		WriteIopsMax: WIM,
	}
	name, vol, cleanf := createVolume(t, client, v)
	defer cleanf()
	if vol.Name != name {
		t.Fatalf("Created volume name did not match request name: [%s] != [%s]\n", vol.Name, name)
	}
	if vol.WriteIopsMax != WIM {
		t.Fatalf("WriteIopsMax did not match request amount: [%d] != [%d]\n", vol.WriteIopsMax, WIM)
	}
}

func TestListVolumes(t *testing.T) {
	client := getClient(t)
	v := &VolOpts{
		Size:         5,
		Replica:      1,
		WriteIopsMax: WIM,
	}
	name, _, cleanf := createVolume(t, client, v)
	defer cleanf()
	vols, err := client.ListVolumes()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, vol := range vols {
		if vol.Name == name {
			found = true
		}
	}
	if !found {
		t.Fatalf("Did not find AppInstance created by test: %s", name)
	}
}

func TestVolumeMetadata(t *testing.T) {
	client := getClient(t)
	v := &VolOpts{
		Size:         5,
		Replica:      1,
		WriteIopsMax: WIM,
	}
	_, vol, cleanf := createVolume(t, client, v)
	defer cleanf()
	m := map[string]string{"my-test": "metadata"}
	m2, err := vol.SetMetadata(&m)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&m, m2) {
		t.Fatalf("metadata sent and metadata recieved are unequal: [%#v] != [%#v]\n", m, m2)
	}

	m3, err := vol.GetMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&m, m3) {
		t.Fatalf("metadata sent and metadata recieved are unequal: [%#v] != [%#v]\n", m, m3)
	}
}

func TestACL(t *testing.T) {
	client := getClient(t)
	v := &VolOpts{
		Size:         5,
		Replica:      1,
		WriteIopsMax: WIM,
	}
	_, vol, cleanf := createVolume(t, client, v)
	defer cleanf()
	init, err := client.CreateGetInitiator()
	if err != nil {
		t.Fatal(err)
	}
	if err = vol.RegisterAcl(init); err != nil {
		t.Fatal(err)
	}
}
