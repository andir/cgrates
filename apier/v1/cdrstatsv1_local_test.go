/*
Real-time Charging System for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can Storagetribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITH*out ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package v1

import (
	"fmt"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"path"
	"reflect"
	"testing"
	"time"
)

var cdrstCfgPath string
var cdrstCfg *config.CGRConfig
var cdrstRpc *rpc.Client

func TestCDRStatsLclLoadConfig(t *testing.T) {
	if !*testLocal {
		return
	}
	var err error
	cdrstCfgPath = path.Join(*dataDir, "conf", "samples", "cdrstats")
	if cdrstCfg, err = config.NewCGRConfigFromFolder(cfgPath); err != nil {
		t.Error(err)
	}
}

func TestCDRStatsLclInitDataDb(t *testing.T) {
	if !*testLocal {
		return
	}
	if err := engine.InitDataDb(cdrstCfg); err != nil {
		t.Fatal(err)
	}
}

func TestCDRStatsLclStartEngine(t *testing.T) {
	if !*testLocal {
		return
	}
	if _, err := engine.StopStartEngine(cdrstCfgPath, *waitRater); err != nil {
		t.Fatal(err)
	}
}

// Connect rpc client to rater
func TestCDRStatsLclRpcConn(t *testing.T) {
	if !*testLocal {
		return
	}
	var err error
	cdrstRpc, err = jsonrpc.Dial("tcp", cdrstCfg.RPCJSONListen) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		t.Fatal("Could not connect to rater: ", err.Error())
	}
}

func TestCDRStatsLclGetQueueIds(t *testing.T) {
	if !*testLocal {
		return
	}
	var queueIds []string
	eQueueIds := []string{"*default"}
	if err := cdrstRpc.Call("CDRStatsV1.GetQueueIds", "", &queueIds); err != nil {
		t.Error("Calling CDRStatsV1.GetQueueIds, got error: ", err.Error())
	} else if !reflect.DeepEqual(eQueueIds, queueIds) {
		t.Errorf("Expecting: %v, received: %v", eQueueIds, queueIds)
	}
}

func TestCDRStatsLclLoadTariffPlanFromFolder(t *testing.T) {
	if !*testLocal {
		return
	}
	reply := ""
	// Simple test that command is executed without errors
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*dataDir, "tariffplans", "cdrstats")}
	if err := cdrstRpc.Call("ApierV1.LoadTariffPlanFromFolder", attrs, &reply); err != nil {
		t.Error("Got error on ApierV1.LoadTariffPlanFromFolder: ", err.Error())
	} else if reply != "OK" {
		t.Error("Calling ApierV1.LoadTariffPlanFromFolder got reply: ", reply)
	}
	time.Sleep(time.Duration(*waitRater) * time.Millisecond) // Give time for scheduler to execute topups
}

func TestCDRStatsLclGetQueueIds2(t *testing.T) {
	if !*testLocal {
		return
	}
	var queueIds []string
	eQueueIds := []string{"*default", "CDRST3", "CDRST4"}
	if err := cdrstRpc.Call("CDRStatsV1.GetQueueIds", "", &queueIds); err != nil {
		t.Error("Calling CDRStatsV1.GetQueueIds, got error: ", err.Error())
	} else if len(eQueueIds) != len(queueIds) {
		t.Errorf("Expecting: %v, received: %v", eQueueIds, queueIds)
	}
}

func TestCDRStatsLclPostCdrs(t *testing.T) {
	if !*testLocal {
		return
	}
	httpClient := new(http.Client)
	storedCdrs := []*engine.StoredCdr{
		&engine.StoredCdr{CgrId: utils.Sha1("dsafdsafa", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()), OrderId: 123, TOR: utils.VOICE, AccId: "dsafdsaf", CdrHost: "192.168.1.1", CdrSource: "test",
			ReqType: utils.META_RATED, Direction: "*out", Tenant: "cgrates.org",
			Category: "call", Account: "1001", Subject: "1001", Destination: "+4986517174963", SetupTime: time.Now(),
			AnswerTime: time.Now(), MediationRunId: utils.DEFAULT_RUNID,
			Usage: time.Duration(10) * time.Second, ExtraFields: map[string]string{"field_extr1": "val_extr1", "fieldextr2": "valextr2"},
			Cost: 1.01, RatedAccount: "dan", RatedSubject: "dan",
		},
		&engine.StoredCdr{CgrId: utils.Sha1("dsafdsafb", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()), OrderId: 123, TOR: utils.VOICE, AccId: "dsafdsaf", CdrHost: "192.168.1.1", CdrSource: "test",
			ReqType: utils.META_RATED, Direction: "*out", Tenant: "cgrates.org",
			Category: "call", Account: "1001", Subject: "1001", Destination: "+4986517174963", SetupTime: time.Now(),
			AnswerTime: time.Now(), MediationRunId: utils.DEFAULT_RUNID,
			Usage: time.Duration(5) * time.Second, ExtraFields: map[string]string{"field_extr1": "val_extr1", "fieldextr2": "valextr2"}, Cost: 1.01, RatedAccount: "dan", RatedSubject: "dan",
		},
		&engine.StoredCdr{CgrId: utils.Sha1("dsafdsafc", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()), OrderId: 123, TOR: utils.VOICE, AccId: "dsafdsaf", CdrHost: "192.168.1.1", CdrSource: "test",
			ReqType: utils.META_RATED, Direction: "*out", Tenant: "cgrates.org",
			Category: "call", Account: "1001", Subject: "1001", Destination: "+4986517174963", SetupTime: time.Now(), AnswerTime: time.Now(),
			MediationRunId: utils.DEFAULT_RUNID,
			Usage:          time.Duration(30) * time.Second, ExtraFields: map[string]string{"field_extr1": "val_extr1", "fieldextr2": "valextr2"}, Cost: 1.01, RatedAccount: "dan", RatedSubject: "dan",
		},
		&engine.StoredCdr{CgrId: utils.Sha1("dsafdsafd", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()), OrderId: 123, TOR: utils.VOICE, AccId: "dsafdsaf", CdrHost: "192.168.1.1", CdrSource: "test",
			ReqType: utils.META_RATED, Direction: "*out", Tenant: "cgrates.org",
			Category: "call", Account: "1001", Subject: "1001", Destination: "+4986517174963", SetupTime: time.Now(), AnswerTime: time.Time{},
			MediationRunId: utils.DEFAULT_RUNID,
			Usage:          time.Duration(0) * time.Second, ExtraFields: map[string]string{"field_extr1": "val_extr1", "fieldextr2": "valextr2"}, Cost: 1.01, RatedAccount: "dan", RatedSubject: "dan",
		},
	}
	for _, storedCdr := range storedCdrs {
		if _, err := httpClient.PostForm(fmt.Sprintf("http://%s/cdr_post", "127.0.0.1:2080"), storedCdr.AsHttpForm()); err != nil {
			t.Error(err.Error())
		}
	}
	time.Sleep(time.Duration(*waitRater) * time.Millisecond)

}

func TestCDRStatsLclGetMetrics1(t *testing.T) {
	if !*testLocal {
		return
	}
	var rcvMetrics1 map[string]float64
	expectedMetrics1 := map[string]float64{"ASR": 75, "ACD": 15, "ACC": 15}
	if err := cdrstRpc.Call("CDRStatsV1.GetMetrics", AttrGetMetrics{StatsQueueId: "*default"}, &rcvMetrics1); err != nil {
		t.Error("Calling CDRStatsV1.GetMetrics, got error: ", err.Error())
	} else if !reflect.DeepEqual(expectedMetrics1, rcvMetrics1) {
		t.Errorf("Expecting: %v, received: %v", expectedMetrics1, rcvMetrics1)
	}
	var rcvMetrics2 map[string]float64
	expectedMetrics2 := map[string]float64{"ASR": 75, "ACD": 15}
	if err := cdrstRpc.Call("CDRStatsV1.GetMetrics", AttrGetMetrics{StatsQueueId: "CDRST4"}, &rcvMetrics2); err != nil {
		t.Error("Calling CDRStatsV1.GetMetrics, got error: ", err.Error())
	} else if !reflect.DeepEqual(expectedMetrics2, rcvMetrics2) {
		t.Errorf("Expecting: %v, received: %v", expectedMetrics2, rcvMetrics2)
	}
}

func TestCDRStatsLclResetMetrics(t *testing.T) {
	if !*testLocal {
		return
	}
	var reply string
	if err := cdrstRpc.Call("CDRStatsV1.ResetQueues", utils.AttrCDRStatsReloadQueues{StatsQueueIds: []string{"CDRST4"}}, &reply); err != nil {
		t.Error("Calling CDRStatsV1.ResetQueues, got error: ", err.Error())
	} else if reply != utils.OK {
		t.Error("Unexpected reply received: ", reply)
	}
	time.Sleep(time.Duration(*waitRater) * time.Millisecond)
	var rcvMetrics1 map[string]float64
	expectedMetrics1 := map[string]float64{"ASR": 75, "ACD": 15, "ACC": 15}
	if err := cdrstRpc.Call("CDRStatsV1.GetMetrics", AttrGetMetrics{StatsQueueId: "*default"}, &rcvMetrics1); err != nil {
		t.Error("Calling CDRStatsV1.GetMetrics, got error: ", err.Error())
	} else if !reflect.DeepEqual(expectedMetrics1, rcvMetrics1) {
		t.Errorf("Expecting: %v, received: %v", expectedMetrics1, rcvMetrics1)
	}
	var rcvMetrics2 map[string]float64
	expectedMetrics2 := map[string]float64{"ASR": -1, "ACD": -1}
	if err := cdrstRpc.Call("CDRStatsV1.GetMetrics", AttrGetMetrics{StatsQueueId: "CDRST4"}, &rcvMetrics2); err != nil {
		t.Error("Calling CDRStatsV1.GetMetrics, got error: ", err.Error())
	} else if !reflect.DeepEqual(expectedMetrics2, rcvMetrics2) {
		t.Errorf("Expecting: %v, received: %v", expectedMetrics2, rcvMetrics2)
	}
}
