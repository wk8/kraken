// Copyright (c) 2016-2019 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package registrybackend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pressly/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/kraken/core"
	"github.com/uber/kraken/lib/backend/backenderrors"
	"github.com/uber/kraken/utils/memsize"
	"github.com/uber/kraken/utils/randutil"
	"github.com/uber/kraken/utils/testutil"
)

func TestClientFactory(t *testing.T) {
	require := require.New(t)

	config := Config{}
	f := blobClientFactory{}
	_, err := f.Create(config, nil)
	require.NoError(err)
}

func TestBlobDownloadBlobSuccess(t *testing.T) {
	require := require.New(t)

	blob := randutil.Blob(32 * memsize.KB)
	namespace := core.NamespaceFixture()

	r := chi.NewRouter()
	r.Get(fmt.Sprintf("/v2/%s/blobs/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		_, err := io.Copy(w, bytes.NewReader(blob))
		require.NoError(err)
	})
	r.Head(fmt.Sprintf("/v2/%s/blobs/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(blob)))
	})
	addr, stop := testutil.StartServer(r)
	defer stop()

	config := newTestConfig(addr)
	client, err := NewBlobClient(config)
	require.NoError(err)

	info, err := client.Stat(namespace, "data")
	require.NoError(err)
	require.Equal(int64(len(blob)), info.Size)

	var b bytes.Buffer
	require.NoError(client.Download(namespace, "data", &b))
	require.Equal(blob, b.Bytes())
}

func TestBlobDownloadManifestSuccess(t *testing.T) {
	require := require.New(t)

	blob := randutil.Blob(32 * memsize.KB)
	namespace := core.NamespaceFixture()

	r := chi.NewRouter()
	r.Get(fmt.Sprintf("/v2/%s/manifests/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		_, err := io.Copy(w, bytes.NewReader(blob))
		require.NoError(err)
	})
	r.Head(fmt.Sprintf("/v2/%s/manifests/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(blob)))
	})
	addr, stop := testutil.StartServer(r)
	defer stop()

	config := newTestConfig(addr)
	client, err := NewBlobClient(config)
	require.NoError(err)

	info, err := client.Stat(namespace, "data")
	require.NoError(err)
	require.Equal(int64(len(blob)), info.Size)

	var b bytes.Buffer
	require.NoError(client.Download(namespace, "data", &b))
	require.Equal(blob, b.Bytes())
}

func TestBlobDownloadFileNotFound(t *testing.T) {
	require := require.New(t)

	namespace := core.NamespaceFixture()

	r := chi.NewRouter()
	r.Get(fmt.Sprintf("/v2/%s/blobs/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("file not found"))
	})
	r.Head(fmt.Sprintf("/v2/%s/blobs/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("file not found"))
	})
	addr, stop := testutil.StartServer(r)
	defer stop()

	config := newTestConfig(addr)
	client, err := NewBlobClient(config)
	require.NoError(err)

	_, err = client.Stat(namespace, "data")
	require.Equal(backenderrors.ErrBlobNotFound, err)

	var b bytes.Buffer
	require.Equal(backenderrors.ErrBlobNotFound, client.Download(namespace, "data", &b))
}

func TestBlobDownloadHeaderTimeout(t *testing.T) {
	require := require.New(t)

	blob := randutil.Blob(32 * memsize.KB)
	namespace := core.NamespaceFixture()

	r := chi.NewRouter()
	r.Get(fmt.Sprintf("/v2/%s/blobs/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Second)
		_, err := io.Copy(w, bytes.NewReader(blob))
		require.NoError(err)
	})
	r.Head(fmt.Sprintf("/v2/%s/blobs/{blob}", namespace), func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Second)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(blob)))
	})
	addr, stop := testutil.StartServer(r)
	defer stop()

	config := newTestConfig(addr)
	config.ResponseHeaderTimeout = 100 * time.Millisecond
	client, err := NewBlobClient(config)
	require.NoError(err)

	_, err = client.Stat(namespace, "data")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "timeout awaiting response headers")
	}

	var b bytes.Buffer
	err = client.Download(namespace, "data", &b)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "timeout awaiting response headers")
	}
}

func TestBlobDownloadConnectTimeout(t *testing.T) {
	require := require.New(t)

	// unroutable address, courtesy of https://stackoverflow.com/a/904609/4867444
	config := newTestConfig("10.255.255.1")
	config.ConnectTimeout = 100 * time.Millisecond
	client, err := NewBlobClient(config)
	require.NoError(err)

	_, err = client.Stat("dummynamespace", "data")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "i/o timeout")
	}

	var b bytes.Buffer
	err = client.Download("dummynamespace", "data", &b)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "i/o timeout")
	}
}
