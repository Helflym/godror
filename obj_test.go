// Copyright 2017 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package goracle

import (
	"context"
	"database/sql/driver"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
)

var (
	testDrv                      *drv
	testCon                      *conn
	clientVersion, serverVersion VersionInfo
)

func init() {
	var err error
	if testDrv, err = newDrv(); err != nil {
		panic(err)
	}
	clientVersion, err = testDrv.ClientVersion()
	if err != nil {
		panic(err)
	}
	fmt.Println("client:", clientVersion)
	dc, err := testDrv.Open(
		fmt.Sprintf("oracle://%s:%s@%s/?poolMinSessions=1&poolMaxSessions=4&poolIncrement=1&connectionClass=POOLED",
			os.Getenv("GORACLE_DRV_TEST_USERNAME"),
			os.Getenv("GORACLE_DRV_TEST_PASSWORD"),
			os.Getenv("GORACLE_DRV_TEST_DB"),
		),
	)
	if err != nil {
		panic(err)
	}
	testCon = dc.(*conn)
	serverVersion, err = testCon.ServerVersion()
	if err != nil {
		panic(err)
	}
	fmt.Println("server:", serverVersion)
}

func TestObjectDirect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	qry := `CREATE OR REPLACE PACKAGE test_pkg_obj IS
  TYPE int_tab_typ IS TABLE OF PLS_INTEGER INDEX BY PLS_INTEGER;
  TYPE rec_typ IS RECORD (int PLS_INTEGER, num NUMBER, vc VARCHAR2(1000), c CHAR(1000), dt DATE);
  TYPE tab_typ IS TABLE OF rec_typ INDEX BY PLS_INTEGER;
END;`
	if err := prepExec(ctx, qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer prepExec(ctx, "DROP PACKAGE test_pkg_obj")

	//defer tl.enableLogging(t)()
	ot, err := testCon.GetObjectType("test_pkg_obj.tab_typ")
	if err != nil {
		if clientVersion.Version >= 12 && serverVersion.Version >= 12 {
			t.Fatal(fmt.Sprintf("%+v", err))
		}
		t.Log(err)
		t.Skip("client or server < 12")
	}
	t.Log(ot)
}

func prepExec(ctx context.Context, qry string, args ...driver.NamedValue) error {
	stmt, err := testCon.PrepareContext(ctx, qry)
	if err != nil {
		return errors.Wrap(err, qry)
	}
	defer stmt.Close()
	st := stmt.(*statement)
	_, err = st.ExecContext(ctx, args)
	return err
}
