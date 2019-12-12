package identity

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestClient_ResolveReference(t *testing.T) {
	type fields struct {
		CaURL *url.URL
	}
	type args struct {
		ref *url.URL
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *url.URL
	}{
		{"ok", fields{&url.URL{Scheme: "https", Host: "localhost"}}, args{&url.URL{Path: "/foo"}}, &url.URL{Scheme: "https", Host: "localhost", Path: "/foo"}},
		{"ok", fields{&url.URL{Scheme: "https", Host: "localhost", Path: "/bar"}}, args{&url.URL{Path: "/foo"}}, &url.URL{Scheme: "https", Host: "localhost", Path: "/foo"}},
		{"ok", fields{&url.URL{Scheme: "https", Host: "localhost"}}, args{&url.URL{Path: "/foo", RawQuery: "foo=bar"}}, &url.URL{Scheme: "https", Host: "localhost", Path: "/foo", RawQuery: "foo=bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				CaURL: tt.fields.CaURL,
			}
			if got := c.ResolveReference(tt.args.ref); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.ResolveReference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadClient(t *testing.T) {
	oldIdentityFile := IdentityFile
	oldDefaultsFile := DefaultsFile
	defer func() {
		IdentityFile = oldIdentityFile
		DefaultsFile = oldDefaultsFile
	}()

	crt, err := tls.LoadX509KeyPair("testdata/identity/identity.crt", "testdata/identity/identity_key")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadFile("testdata/certs/root_ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(b)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		Certificates: []tls.Certificate{crt},
		RootCAs:      pool,
	}
	expected := &Client{
		CaURL: &url.URL{Scheme: "https", Host: "127.0.0.1"},
		Client: &http.Client{
			Transport: tr,
		},
	}

	tests := []struct {
		name    string
		prepare func()
		want    *Client
		wantErr bool
	}{
		{"ok", func() { IdentityFile = "testdata/config/identity.json"; DefaultsFile = "testdata/config/defaults.json" }, expected, false},
		{"fail identity", func() { IdentityFile = "testdata/config/missing.json"; DefaultsFile = "testdata/config/defaults.json" }, nil, true},
		{"fail identity", func() { IdentityFile = "testdata/config/fail.json"; DefaultsFile = "testdata/config/defaults.json" }, nil, true},
		{"fail defaults", func() { IdentityFile = "testdata/config/identity.json"; DefaultsFile = "testdata/config/missing.json" }, nil, true},
		{"fail defaults", func() { IdentityFile = "testdata/config/identity.json"; DefaultsFile = "testdata/config/fail.json" }, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			got, err := LoadClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("LoadClient() = %#v, want %#v", got, tt.want)
				}
			} else {
				if !reflect.DeepEqual(got.CaURL, tt.want.CaURL) ||
					!reflect.DeepEqual(got.Client.Transport.(*http.Transport).TLSClientConfig.RootCAs, tt.want.Client.Transport.(*http.Transport).TLSClientConfig.RootCAs) ||
					!reflect.DeepEqual(got.Client.Transport.(*http.Transport).TLSClientConfig.Certificates, tt.want.Client.Transport.(*http.Transport).TLSClientConfig.Certificates) {
					t.Errorf("LoadClient() = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func Test_defaultsConfig_Validate(t *testing.T) {
	type fields struct {
		CaURL string
		Root  string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"ok", fields{"https://127.0.0.1", "root_ca.crt"}, false},
		{"fail ca-url", fields{"", "root_ca.crt"}, true},
		{"fail root", fields{"https://127.0.0.1", ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultsConfig{
				CaURL: tt.fields.CaURL,
				Root:  tt.fields.Root,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("defaultsConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
