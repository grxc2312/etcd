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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/tests/v3/framework/config"
	"go.etcd.io/etcd/tests/v3/framework/testutils"
)

type txnReq struct {
	compare  []string
	ifSucess []string
	ifFail   []string
	results  []string
}

func TestTxnSucc(t *testing.T) {
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
	reqs := []txnReq{
		{
			compare:  []string{`value("key1") != "value2"`, `value("key2") != "value1"`},
			ifSucess: []string{"get key1", "get key2"},
			results:  []string{"SUCCESS", "key1", "value1", "key2", "value2"},
		},
		{
			compare:  []string{`version("key1") = "1"`, `version("key2") = "1"`},
			ifSucess: []string{"get key1", "get key2", `put "key \"with\" space" "value \x23"`},
			ifFail:   []string{`put key1 "fail"`, `put key2 "fail"`},
			results:  []string{"SUCCESS", "key1", "value1", "key2", "value2", "OK"},
		},
		{
			compare:  []string{`version("key \"with\" space") = "1"`},
			ifSucess: []string{`get "key \"with\" space"`},
			results:  []string{"SUCCESS", `key "with" space`, "value \x23"},
		},
	}
	testRunner.BeforeTest(t)
	for _, cfg := range tcs {
		t.Run(cfg.name, func(t *testing.T) {
			clus := testRunner.NewCluster(t, cfg.config)
			defer clus.Close()
			cc := clus.Client()
			testutils.ExecuteWithTimeout(t, 10*time.Second, func() {
				if err := cc.Put("key1", "value1", config.PutOptions{}); err != nil {
					t.Fatalf("could not create key:%s, value:%s", "key1", "value1")
				}
				if err := cc.Put("key2", "value2", config.PutOptions{}); err != nil {
					t.Fatalf("could not create key:%s, value:%s", "key2", "value2")
				}
				for _, req := range reqs {
					resp, err := cc.Txn(req.compare, req.ifSucess, req.ifFail, config.TxnOptions{
						Interactive: true,
					})
					if err != nil {
						t.Errorf("Txn returned error: %s", err)
					}
					assert.Equal(t, req.results, getRespValues(resp))
				}
			})
		})
	}
}

func TestTxnFail(t *testing.T) {
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
	reqs := []txnReq{
		{
			compare:  []string{`version("key") < "0"`},
			ifSucess: []string{`put key "success"`},
			ifFail:   []string{`put key "fail"`},
			results:  []string{"FAILURE", "OK"},
		},
		{
			compare:  []string{`value("key1") != "value1"`},
			ifSucess: []string{`put key1 "success"`},
			ifFail:   []string{`put key1 "fail"`},
			results:  []string{"FAILURE", "OK"},
		},
	}
	testRunner.BeforeTest(t)
	for _, cfg := range tcs {
		t.Run(cfg.name, func(t *testing.T) {
			clus := testRunner.NewCluster(t, cfg.config)
			defer clus.Close()
			cc := clus.Client()
			testutils.ExecuteWithTimeout(t, 10*time.Second, func() {
				if err := cc.Put("key1", "value1", config.PutOptions{}); err != nil {
					t.Fatalf("could not create key:%s, value:%s", "key1", "value1")
				}
				for _, req := range reqs {
					resp, err := cc.Txn(req.compare, req.ifSucess, req.ifFail, config.TxnOptions{
						Interactive: true,
					})
					if err != nil {
						t.Errorf("Txn returned error: %s", err)
					}
					assert.Equal(t, req.results, getRespValues(resp))
				}
			})
		})
	}
}

func getRespValues(r *clientv3.TxnResponse) []string {
	ss := []string{}
	if r.Succeeded {
		ss = append(ss, "SUCCESS")
	} else {
		ss = append(ss, "FAILURE")
	}
	for _, resp := range r.Responses {
		switch v := resp.Response.(type) {
		case *pb.ResponseOp_ResponseDeleteRange:
			r := (clientv3.DeleteResponse)(*v.ResponseDeleteRange)
			ss = append(ss, fmt.Sprintf("%d", r.Deleted))
		case *pb.ResponseOp_ResponsePut:
			r := (clientv3.PutResponse)(*v.ResponsePut)
			ss = append(ss, "OK")
			if r.PrevKv != nil {
				ss = append(ss, string(r.PrevKv.Key), string(r.PrevKv.Value))
			}
		case *pb.ResponseOp_ResponseRange:
			r := (clientv3.GetResponse)(*v.ResponseRange)
			for _, kv := range r.Kvs {
				ss = append(ss, string(kv.Key), string(kv.Value))
			}
		default:
			ss = append(ss, fmt.Sprintf("\"Unknown\" : %q\n", fmt.Sprintf("%+v", v)))
		}
	}
	return ss
}
