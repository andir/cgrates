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

package v2

import (
	"flag"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os/exec"
	"path"
	"testing"
	"time"
)

var testLocal = flag.Bool("local", false, "Perform the tests only on local test environment, not by default.") // This flag will be passed here via "go test -local" args
var dataDir = flag.String("data_dir", "/usr/share/cgrates", "CGR data dir path here")
var waitRater = flag.Int("wait_rater", 500, "Number of miliseconds to wait for rater to start and cache")

var cdrsCfgPath string
var cdrsCfg *config.CGRConfig
var cdrsRpc *rpc.Client

var cmdEngineCdrsMysql *exec.Cmd

func TestInitConfig(t *testing.T) {
	if !*testLocal {
		return
	}
	var err error
	cdrsCfgPath = path.Join(*dataDir, "conf", "samples", "cdrsv2mysql_local_test.cfg")
	cdrsCfg, err = config.NewCGRConfigFromFile(&cdrsCfgPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestV2CdrsMysqlInitDataDb(t *testing.T) {
	if !*testLocal {
		return
	}
	if err := engine.InitDataDb(cdrsCfg); err != nil {
		t.Fatal(err)
	}
}

// InitDb so we can rely on count
func TestV2CdrsMysqlInitCdrDb(t *testing.T) {
	if !*testLocal {
		return
	}
	if err := engine.InitCdrDb(cdrsCfg); err != nil {
		t.Fatal(err)
	}
}

func TestV2CdrsMysqlStartEngine(t *testing.T) {
	if !*testLocal {
		return
	}
	var err error
	if cmdEngineCdrsMysql, err = engine.StartEngine(cdrsCfgPath, *waitRater); err != nil {
		t.Fatal(err)
	}
}

// Connect rpc client to rater
func TestV2CdrsMysqlRpcConn(t *testing.T) {
	if !*testLocal {
		return
	}
	var err error
	cdrsRpc, err = jsonrpc.Dial("tcp", cdrsCfg.RPCJSONListen) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		t.Fatal("Could not connect to rater: ", err.Error())
	}
}

// Insert some CDRs
func TestV2CdrsMysqlProcessCdr(t *testing.T) {
	if !*testLocal {
		return
	}
	var reply string
	cdrs := []*utils.StoredCdr{
		&utils.StoredCdr{CgrId: utils.Sha1("dsafdsaf", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()), OrderId: 123, TOR: utils.VOICE, AccId: "dsafdsaf",
			CdrHost: "192.168.1.1", CdrSource: "test", ReqType: "rated", Direction: "*out", Tenant: "cgrates.org", Category: "call", Account: "1001", Subject: "1001", Destination: "1002",
			SetupTime: time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC), AnswerTime: time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC), MediationRunId: utils.DEFAULT_RUNID,
			Usage: time.Duration(10) * time.Second, ExtraFields: map[string]string{"field_extr1": "val_extr1", "fieldextr2": "valextr2"}, Cost: 1.01, RatedAccount: "dan", RatedSubject: "dans",
		},
		&utils.StoredCdr{CgrId: utils.Sha1("abcdeftg", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()), OrderId: 123, TOR: utils.VOICE, AccId: "dsafdsaf",
			CdrHost: "192.168.1.1", CdrSource: "test", ReqType: "rated", Direction: "*out", Tenant: "cgrates.org", Category: "call", Account: "1002", Subject: "1002", Destination: "1002",
			SetupTime: time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC), AnswerTime: time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC), MediationRunId: utils.DEFAULT_RUNID,
			Usage: time.Duration(10) * time.Second, ExtraFields: map[string]string{"field_extr1": "val_extr1", "fieldextr2": "valextr2"}, Cost: 1.01, RatedAccount: "dan", RatedSubject: "dans",
		},
		&utils.StoredCdr{CgrId: utils.Sha1("aererfddf", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()), OrderId: 123, TOR: utils.VOICE, AccId: "dsafdsaf",
			CdrHost: "192.168.1.1", CdrSource: "test", ReqType: "rated", Direction: "*out", Tenant: "cgrates.org", Category: "call", Account: "1003", Subject: "1003", Destination: "1002",
			SetupTime: time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC), AnswerTime: time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC), MediationRunId: utils.DEFAULT_RUNID,
			Usage: time.Duration(10) * time.Second, ExtraFields: map[string]string{"field_extr1": "val_extr1", "fieldextr2": "valextr2"}, Cost: 1.01, RatedAccount: "dan", RatedSubject: "dans",
		},
	}
	for _, cdr := range cdrs {
		if err := cdrsRpc.Call("CdrsV2.ProcessCdr", cdr, &reply); err != nil {
			t.Error("Unexpected error: ", err.Error())
		} else if reply != utils.OK {
			t.Error("Unexpected reply received: ", reply)
		}
	}
}

func TestV2CdrsMysqlGetCdrs(t *testing.T) {
	if !*testLocal {
		return
	}
	var reply []*utils.StoredCdr
	req := utils.AttrGetCdrs{}
	if err := cdrsRpc.Call("ApierV2.GetCdrs", req, &reply); err != nil {
		t.Error("Unexpected error: ", err.Error())
	} else if len(reply) != 3 {
		t.Error("Unexpected number of CDRs returned: ", len(reply))
	}
}

/*
func TestV2CdrsMysqlCountCdrs(t *testing.T) {
	if !*testLocal {
		return
	}
	var reply int64
	req := utils.AttrGetCdrs{}
	if err := cdrsRpc.Call("ApierV2.CountCdrs", req, &reply); err != nil {
		t.Error("Unexpected error: ", err.Error())
	} else if reply != 3 {
		t.Error("Unexpected number of CDRs returned: ", reply)
	}
}
*/

func TestV2CdrsMysqlKillEngine(t *testing.T) {
	if !*testLocal {
		return
	}
	if err := engine.KillEngine(*waitRater); err != nil {
		t.Error(err)
	}
}