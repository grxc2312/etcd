// Copyright 2022 The etcd Authors
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

package common

import (
	"strings"
	"testing"
	"time"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/tests/v3/framework/config"
	"go.etcd.io/etcd/tests/v3/framework/testutils"
)

func TestRoleAdd_Simple(t *testing.T) {
	testRunner.BeforeTest(t)
	tcs := []struct {
		name   string
		config config.ClusterConfig
	}{
		{
			name:   "NoTLS",
			config: config.ClusterConfig{ClusterSize: 1},
		},
		{
			name:   "PeerTLS",
			config: config.ClusterConfig{ClusterSize: 3, PeerTLS: config.ManualTLS},
		},
		{
			name:   "PeerAutoTLS",
			config: config.ClusterConfig{ClusterSize: 3, PeerTLS: config.AutoTLS},
		},
		{
			name:   "ClientTLS",
			config: config.ClusterConfig{ClusterSize: 1, ClientTLS: config.ManualTLS},
		},
		{
			name:   "ClientAutoTLS",
			config: config.ClusterConfig{ClusterSize: 1, ClientTLS: config.AutoTLS},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			clus := testRunner.NewCluster(t, tc.config)
			defer clus.Close()
			cc := clus.Client()

			testutils.ExecuteWithTimeout(t, 10*time.Second, func() {
				_, err := cc.RoleAdd("root")
				if err != nil {
					t.Fatalf("want no error, but got (%v)", err)
				}
			})
		})
	}
}

func TestRoleAdd_Error(t *testing.T) {
	testRunner.BeforeTest(t)
	clus := testRunner.NewCluster(t, config.ClusterConfig{ClusterSize: 1})
	defer clus.Close()
	cc := clus.Client()
	testutils.ExecuteWithTimeout(t, 10*time.Second, func() {
		_, err := cc.RoleAdd("test-role")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		_, err = cc.RoleAdd("test-role")
		if err == nil || !strings.Contains(err.Error(), rpctypes.ErrRoleAlreadyExist.Error()) {
			t.Fatalf("want (%v) error, but got (%v)", rpctypes.ErrRoleAlreadyExist, err)
		}
		_, err = cc.RoleAdd("")
		if err == nil || !strings.Contains(err.Error(), rpctypes.ErrRoleEmpty.Error()) {
			t.Fatalf("want (%v) error, but got (%v)", rpctypes.ErrRoleEmpty, err)
		}
	})
}

func TestRootRole(t *testing.T) {
	testRunner.BeforeTest(t)
	clus := testRunner.NewCluster(t, config.ClusterConfig{ClusterSize: 1})
	defer clus.Close()
	cc := clus.Client()
	testutils.ExecuteWithTimeout(t, 10*time.Second, func() {
		_, err := cc.RoleAdd("root")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		resp, err := cc.RoleGet("root")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		t.Logf("get role resp %+v", resp)
		// granting to root should be refused by server and a no-op
		_, err = cc.RoleGrantPermission("root", "foo", "", clientv3.PermissionType(clientv3.PermReadWrite))
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		resp2, err := cc.RoleGet("root")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		t.Logf("get role resp %+v", resp2)
	})
}

func TestRoleGrantRevokePermission(t *testing.T) {
	testRunner.BeforeTest(t)
	clus := testRunner.NewCluster(t, config.ClusterConfig{ClusterSize: 1})
	defer clus.Close()
	cc := clus.Client()
	testutils.ExecuteWithTimeout(t, 10*time.Second, func() {
		_, err := cc.RoleAdd("role1")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		_, err = cc.RoleGrantPermission("role1", "bar", "", clientv3.PermissionType(clientv3.PermRead))
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		_, err = cc.RoleGrantPermission("role1", "bar", "", clientv3.PermissionType(clientv3.PermWrite))
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		_, err = cc.RoleGrantPermission("role1", "bar", "foo", clientv3.PermissionType(clientv3.PermReadWrite))
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		_, err = cc.RoleRevokePermission("role1", "foo", "")
		if err == nil || !strings.Contains(err.Error(), rpctypes.ErrPermissionNotGranted.Error()) {
			t.Fatalf("want error (%v), but got (%v)", rpctypes.ErrPermissionNotGranted, err)
		}
		_, err = cc.RoleRevokePermission("role1", "bar", "foo")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
	})
}

func TestRoleDelete(t *testing.T) {
	testRunner.BeforeTest(t)
	clus := testRunner.NewCluster(t, config.ClusterConfig{ClusterSize: 1})
	defer clus.Close()
	cc := clus.Client()
	testutils.ExecuteWithTimeout(t, 10*time.Second, func() {
		_, err := cc.RoleAdd("role1")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
		_, err = cc.RoleDelete("role1")
		if err != nil {
			t.Fatalf("want no error, but got (%v)", err)
		}
	})
}
