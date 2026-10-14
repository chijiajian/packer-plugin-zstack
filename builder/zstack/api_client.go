// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/util/jsonutils"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

type apiError struct {
	Code       string   `json:"code"`
	FormatArgs []string `json:"formatArgs"`
	raw        string
}

func (e *apiError) Error() string { return "ZStack API: " + e.raw }

// SDK v0.0.10's httputils.requestInternal ignores its context argument.
// Keep SDK wire models and date decoding, but own the small HTTP boundary so
// cancellation, bounded polling and request-log redaction are enforceable.
type apiClient struct {
	base                       *url.URL
	http                       *http.Client
	accessKey, secret, session string
	account, password          string
	pollInterval               time.Duration
}

func newAPIClient(c *AccessConfig) (*apiClient, error) {
	host := c.Host
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	u, err := url.Parse(host)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("zstack_host must be an HTTP(S) host without credentials or query parameters")
	}
	if u.Path != "" && u.Path != "/" && u.Path != "/zstack" {
		return nil, fmt.Errorf("zstack_host must not contain a custom path")
	}
	if c.Port != 0 {
		u.Host = net.JoinHostPort(u.Hostname(), strconv.Itoa(c.Port))
	} else if u.Port() == "" && u.Scheme == "http" {
		u.Host = net.JoinHostPort(u.Hostname(), "8080")
	}
	u.Path = "/zstack/"
	return &apiClient{base: u, http: &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}, accessKey: c.AccessKeyId, secret: c.AccessKeySecret, account: c.AccountName, password: c.AccountPassword, pollInterval: 2 * time.Second}, nil
}
func (a *apiClient) Login(ctx context.Context) (*view.SessionInventoryView, error) {
	var out view.SessionInventoryView
	params := map[string]any{"logInByAccount": map[string]any{"accountName": a.account, "password": fmt.Sprintf("%x", sha512.Sum512([]byte(a.password)))}}
	if err := a.Put(ctx, "v1/accounts/login", "", params, &out); err != nil {
		return nil, err
	}
	if out.Uuid == "" {
		return nil, fmt.Errorf("login returned empty session")
	}
	a.session = out.Uuid
	return &out, nil
}
func (a *apiClient) do(ctx context.Context, method, resource string, values url.Values, body any) (jsonutils.JSONObject, error) {
	u := *a.base
	u.Path += strings.TrimPrefix(resource, "/")
	u.RawQuery = values.Encode()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	for {
		req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if a.accessKey != "" {
			date := time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST")
			sign := hmac.New(sha1.New, []byte(a.secret))
			fmt.Fprintf(sign, "%s\n%s\n%s", method, date, strings.TrimPrefix(u.EscapedPath(), "/zstack"))
			req.Header.Set("Date", date)
			req.Header.Set("Authorization", "ZStack "+a.accessKey+":"+base64.StdEncoding.EncodeToString(sign.Sum(nil)))
		} else if a.session != "" {
			req.Header.Set("Authorization", "OAuth "+a.session)
		}
		resp, err := a.http.Do(req)
		if err != nil {
			return nil, err
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			raw = []byte(`{}`)
		}
		data, err := jsonutils.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("ZStack returned invalid JSON (HTTP %d)", resp.StatusCode)
		}
		if e, err := data.Get("error"); err == nil && e.String() != "null" {
			apiErr := &apiError{raw: e.String()}
			if err := json.Unmarshal([]byte(e.String()), apiErr); err != nil {
				return nil, fmt.Errorf("ZStack API: %s", e.String())
			}
			return nil, apiErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("ZStack HTTP %d: %s", resp.StatusCode, data.String())
		}
		location, _ := data.GetString("location")
		if location == "" {
			location = resp.Header.Get("Location")
		}
		if resp.StatusCode != http.StatusAccepted && location == "" {
			return data, nil
		}
		if location != "" {
			callback, e := url.Parse(location)
			if e != nil {
				return nil, fmt.Errorf("invalid ZStack job location")
			}
			if !strings.HasPrefix(callback.Path, "/zstack/v1/api-jobs/") && !strings.HasPrefix(callback.Path, "/v1/api-jobs/") {
				return nil, fmt.Errorf("unexpected ZStack job location path")
			}
			// Some management nodes return their internal address. Poll the configured
			// origin, never send credentials to the authority of a response URL.
			u.Path = callback.Path
			if strings.HasPrefix(u.Path, "/v1/") {
				u.Path = "/zstack" + u.Path
			}
			u.RawQuery = callback.RawQuery
		} else if method != http.MethodGet {
			return nil, fmt.Errorf("asynchronous response missing job location")
		}
		method = http.MethodGet
		payload = nil
		timer := time.NewTimer(a.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
func (a *apiClient) List(ctx context.Context, resource string, params *param.QueryParam, out any) error {
	data, err := a.do(ctx, http.MethodGet, resource, params.Values, nil)
	if err != nil {
		return err
	}
	return data.Unmarshal(out, "inventories")
}
func (a *apiClient) Get(ctx context.Context, resource, id string, params any, out any) error {
	data, err := a.do(ctx, http.MethodGet, resource+"/"+url.PathEscape(id), nil, nil)
	if err != nil {
		return err
	}
	rows, err := data.GetArray("inventories")
	if err != nil {
		return err
	}
	if len(rows) != 1 {
		return fmt.Errorf("resource %s: expected one inventory, got %d", id, len(rows))
	}
	return rows[0].Unmarshal(out)
}
func (a *apiClient) Post(ctx context.Context, resource string, params, out any) error {
	data, err := a.do(ctx, http.MethodPost, resource, nil, params)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return data.Unmarshal(out, "inventory")
}
func (a *apiClient) Put(ctx context.Context, resource, id string, params, out any) error {
	return a.PutWithRespKey(ctx, resource, id, "inventory", params, out)
}
func (a *apiClient) PutWithRespKey(ctx context.Context, resource, id, key string, params, out any) error {
	if id != "" {
		resource += "/" + url.PathEscape(id) + "/actions"
	}
	data, err := a.do(ctx, http.MethodPut, resource, nil, params)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if key == "" {
		return data.Unmarshal(out)
	}
	return data.Unmarshal(out, key)
}
func (a *apiClient) Delete(ctx context.Context, resource, id, mode string) error {
	_, err := a.do(ctx, http.MethodDelete, resource+"/"+url.PathEscape(id), url.Values{"deleteMode": {mode}}, nil)
	return err
}
