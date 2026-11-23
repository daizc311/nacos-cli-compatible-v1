package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBaseURL(t *testing.T) {
	tests := []struct {
		name       string
		serverAddr string
		scheme     string
		want       string
	}{
		{
			name:       "default http scheme",
			serverAddr: "nacos.example.com:8848",
			scheme:     "",
			want:       "http://nacos.example.com:8848",
		},
		{
			name:       "explicit http scheme",
			serverAddr: "nacos.example.com:8848",
			scheme:     "http",
			want:       "http://nacos.example.com:8848",
		},
		{
			name:       "https scheme",
			serverAddr: "nacos.example.com:443",
			scheme:     "https",
			want:       "https://nacos.example.com:443",
		},
		{
			name:       "https without explicit port",
			serverAddr: "nacos.example.com",
			scheme:     "https",
			want:       "https://nacos.example.com",
		},
		{
			name:       "uppercase scheme normalized",
			serverAddr: "nacos.example.com:8848",
			scheme:     "HTTPS",
			want:       "HTTPS://nacos.example.com:8848",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NacosClient{
				ServerAddr: tt.serverAddr,
				Scheme:     tt.scheme,
			}
			got := c.BaseURL()
			if got != tt.want {
				t.Errorf("BaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewNacosClientScheme(t *testing.T) {
	// Test that scheme is properly stored when creating client
	c, err := NewNacosClient(
		"localhost:8848",
		"public",
		AuthTypeNone,
		"", "", "", "", "", "", "",
		"https",
	)
	if err != nil {
		t.Fatalf("NewNacosClient() error = %v", err)
	}
	if c.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", c.Scheme, "https")
	}
	if c.BaseURL() != "https://localhost:8848" {
		t.Errorf("BaseURL() = %q, want %q", c.BaseURL(), "https://localhost:8848")
	}
}

func TestNewNacosClientDefaultScheme(t *testing.T) {
	// Test that empty scheme defaults to "http"
	c, err := NewNacosClient(
		"localhost:8848",
		"public",
		AuthTypeNone,
		"", "", "", "", "", "", "",
		"",
	)
	if err != nil {
		t.Fatalf("NewNacosClient() error = %v", err)
	}
	if c.Scheme != "http" {
		t.Errorf("Scheme = %q, want %q", c.Scheme, "http")
	}
	if c.BaseURL() != "http://localhost:8848" {
		t.Errorf("BaseURL() = %q, want %q", c.BaseURL(), "http://localhost:8848")
	}
}

func TestFetchStsCredentialsSendsClusterIDHeader(t *testing.T) {
	t.Setenv("HICLAW_CLUSTER_ID", "cluster-123")

	stsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer auth-token" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer auth-token")
		}
		if got := r.Header.Get("X-HiClaw-Cluster-ID"); got != "cluster-123" {
			t.Fatalf("X-HiClaw-Cluster-ID header = %q, want %q", got, "cluster-123")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_key_id":"ak","access_key_secret":"sk","security_token":"token","expires_in_sec":600}`))
	}))
	defer stsServer.Close()

	c, err := NewNacosClient(
		"localhost:8848",
		"public",
		AuthTypeStsToken,
		"", "", "", "", "",
		stsServer.URL,
		"auth-token",
		"",
	)
	if err != nil {
		t.Fatalf("NewNacosClient() error = %v", err)
	}
	if c.AccessKey != "ak" || c.SecretKey != "sk" || c.SecurityToken != "token" {
		t.Fatalf("STS credentials = (%q, %q, %q), want (ak, sk, token)", c.AccessKey, c.SecretKey, c.SecurityToken)
	}
}

func TestFetchStsCredentialsSendsAgentTeamsClusterIDHeader(t *testing.T) {
	t.Setenv("AGENTTEAMS_CLUSTER_ID", "agentteams-cluster")
	t.Setenv("HICLAW_CLUSTER_ID", "hiclaw-cluster")

	stsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-AgentTeams-Cluster-ID"); got != "agentteams-cluster" {
			t.Fatalf("X-AgentTeams-Cluster-ID = %q, want %q", got, "agentteams-cluster")
		}
		if got := r.Header.Get("X-HiClaw-Cluster-ID"); got != "" {
			t.Fatalf("X-HiClaw-Cluster-ID = %q, want empty", got)
		}
		_, _ = w.Write([]byte(`{"access_key_id":"ak","access_key_secret":"sk","security_token":"st","expires_in_sec":60}`))
	}))
	defer stsServer.Close()

	if _, err := NewNacosClient(
		"localhost:8848",
		"public",
		AuthTypeStsAgentTeams,
		"", "", "", "", "",
		stsServer.URL,
		"auth-token",
		"http",
	); err != nil {
		t.Fatalf("NewNacosClient() error = %v", err)
	}
}

func TestNacosClientReusesHTTPClientWithTimeout(t *testing.T) {
	c, err := NewNacosClient(
		"127.0.0.1:8848",
		"public",
		AuthTypeNone,
		"", "", "", "", "", "", "",
		"http",
	)
	if err != nil {
		t.Fatal(err)
	}

	first := c.HTTPClient()
	second := c.HTTPClient()
	if first != second {
		t.Fatal("HTTPClient returned different instances")
	}
	if first.Timeout != DefaultHTTPTimeout {
		t.Fatalf("timeout = %s, want %s", first.Timeout, DefaultHTTPTimeout)
	}
}

func TestNewNacosClientWithTokenSetsAuthorizationHeader(t *testing.T) {
	c, err := NewNacosClient(
		"127.0.0.1:8848",
		"public",
		AuthTypeNone,
		"", "", "", "", "", "", "",
		"http",
		WithToken("test-token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthType != AuthTypeToken {
		t.Fatalf("AuthType = %q, want %q", c.AuthType, AuthTypeToken)
	}

	req, err := c.NewAuthedRequest(http.MethodGet, c.BaseURL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("Authorization header = %q, want %q", got, "Bearer test-token")
	}
	if got := c.httpClient.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("resty Authorization header = %q, want %q", got, "Bearer test-token")
	}
}

func clientForTestServer(t *testing.T, rawURL string) *NacosClient {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewNacosClient(
		u.Host,
		"public",
		AuthTypeNone,
		"", "", "", "", "", "", "",
		u.Scheme,
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestGetConfigV1UsesOpenAPI(t *testing.T) {
	const content = `{"code":0,"data":{"content":"inner"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/nacos/v1/cs/configs" {
			t.Errorf("path = %s, want /nacos/v1/cs/configs", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("dataId") != "app.yaml" {
			t.Errorf("dataId = %q", q.Get("dataId"))
		}
		if q.Get("group") != "DEFAULT_GROUP" {
			t.Errorf("group = %q", q.Get("group"))
		}
		if q.Get("tenant") != "ns-1" {
			t.Errorf("tenant = %q", q.Get("tenant"))
		}
		if q.Get("search") != "" || q.Get("groupName") != "" || q.Get("namespaceId") != "" {
			t.Errorf("unexpected v3/list params: %s", r.URL.RawQuery)
		}
		if got := r.Header.Get("Spas-AccessKey"); got != "ak" {
			t.Errorf("Spas-AccessKey = %q, want ak", got)
		}
		_, _ = w.Write([]byte(content))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	c.authLoginVersion = "v1"
	c.AuthType = AuthTypeAliyun
	c.AccessKey = "ak"
	c.SecretKey = "sk"
	c.Namespace = "ns-1"

	got, err := c.GetConfig("app.yaml", "DEFAULT_GROUP")
	if err != nil {
		t.Fatal(err)
	}
	if got != content {
		t.Fatalf("content = %q, want raw v1 body", got)
	}
}

func TestGetConfigV1NacosAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nacos/v1/cs/configs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("accessToken") != "tok" {
			t.Errorf("accessToken = %q", r.URL.Query().Get("accessToken"))
		}
		if r.URL.Query().Get("tenant") != "public" {
			t.Errorf("tenant = %q, want public", r.URL.Query().Get("tenant"))
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want empty", got)
		}
		if got := r.Header.Get("Spas-AccessKey"); got != "" {
			t.Errorf("Spas-AccessKey = %q, want empty", got)
		}
		_, _ = w.Write([]byte("plain"))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	c.authLoginVersion = "v1"
	c.AuthType = AuthTypeNacos
	c.AccessToken = "tok"
	c.Namespace = "public"

	got, err := c.GetConfig("app.yaml", "DEFAULT_GROUP")
	if err != nil {
		t.Fatal(err)
	}
	if got != "plain" {
		t.Fatalf("content = %q, want plain", got)
	}
}

func TestGetConfigV1HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "config data not exist", http.StatusNotFound)
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	c.authLoginVersion = "v1"

	_, err := c.GetConfig("missing", "DEFAULT_GROUP")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get config (v1) failed (404 Not Found)") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestGetConfigUsesV3WhenLoginVersionIsNotV1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nacos/v3/client/cs/config" {
			t.Errorf("path = %s, want v3 client config", r.URL.Path)
		}
		if r.URL.Query().Get("groupName") != "DEFAULT_GROUP" {
			t.Errorf("groupName = %q", r.URL.Query().Get("groupName"))
		}
		if r.URL.Query().Get("namespaceId") != "" {
			t.Errorf("public namespace should be omitted, got %q", r.URL.Query().Get("namespaceId"))
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"content":"v3-body"}}`))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	got, err := c.GetConfig("app.yaml", "DEFAULT_GROUP")
	if err != nil {
		t.Fatal(err)
	}
	if got != "v3-body" {
		t.Fatalf("content = %q, want v3-body", got)
	}
}

func TestPublishConfigV1UsesOpenAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/nacos/v1/cs/configs" {
			t.Errorf("path = %s, want /nacos/v1/cs/configs", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("dataId") != "app.yaml" {
			t.Errorf("dataId = %q", form.Get("dataId"))
		}
		if form.Get("group") != "DEFAULT_GROUP" {
			t.Errorf("group = %q", form.Get("group"))
		}
		if form.Get("content") != "key: value" {
			t.Errorf("content = %q", form.Get("content"))
		}
		if form.Get("tenant") != "ns-1" {
			t.Errorf("tenant = %q", form.Get("tenant"))
		}
		if form.Get("accessToken") != "tok" {
			t.Errorf("accessToken = %q", form.Get("accessToken"))
		}
		if form.Get("groupName") != "" || form.Get("namespaceId") != "" {
			t.Errorf("unexpected v3 params: %s", body)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want empty", got)
		}
		_, _ = w.Write([]byte("true"))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	c.authLoginVersion = "v1"
	c.AuthType = AuthTypeNacos
	c.AccessToken = "tok"
	c.Namespace = "ns-1"

	if err := c.PublishConfig("app.yaml", "DEFAULT_GROUP", "key: value"); err != nil {
		t.Fatal(err)
	}
}

func TestPublishConfigV1False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("false"))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	c.authLoginVersion = "v1"

	err := c.PublishConfig("app.yaml", "DEFAULT_GROUP", "key: value")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "publish config failed: server returned false" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestPublishConfigV1InvalidBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-a-bool"))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	c.authLoginVersion = "v1"

	err := c.PublishConfig("app.yaml", "DEFAULT_GROUP", "key: value")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "publish config failed: invalid response format: not-a-bool" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestPublishConfigV1HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("no right"))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	c.authLoginVersion = "v1"

	err := c.PublishConfig("app.yaml", "DEFAULT_GROUP", "key: value")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "publish config (v1) failed (403 Forbidden)") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestPublishConfigUsesV3WhenLoginVersionIsNotV1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nacos/v3/admin/cs/config" {
			t.Errorf("path = %s, want v3 admin config", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("groupName") != "DEFAULT_GROUP" {
			t.Errorf("groupName = %q", form.Get("groupName"))
		}
		if form.Get("namespaceId") != "public" {
			t.Errorf("namespaceId = %q", form.Get("namespaceId"))
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":true}`))
	}))
	defer srv.Close()

	c := clientForTestServer(t, srv.URL)
	if err := c.PublishConfig("app.yaml", "DEFAULT_GROUP", "key: value"); err != nil {
		t.Fatal(err)
	}
}
