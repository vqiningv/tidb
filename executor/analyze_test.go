// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package executor_test

import (
	"fmt"
	"strings"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb/executor"
	"github.com/pingcap/tidb/model"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/util/testkit"
)

func (s *testSuite) TestAnalyzePartition(c *C) {
	tk := testkit.NewTestKit(c, s.store)
	tk.MustExec("set @@session.tidb_enable_table_partition=1")
	tk.MustExec("use test")
	tk.MustExec("drop table if exists t")
	createTable := `CREATE TABLE t (a int, b int, c varchar(10), primary key(a), index idx(b))
PARTITION BY RANGE ( a ) (
		PARTITION p0 VALUES LESS THAN (6),
		PARTITION p1 VALUES LESS THAN (11),
		PARTITION p2 VALUES LESS THAN (16),
		PARTITION p3 VALUES LESS THAN (21)
)`
	tk.MustExec(createTable)
	for i := 1; i < 21; i++ {
		tk.MustExec(fmt.Sprintf(`insert into t values (%d, %d, "hello")`, i, i))
	}
	tk.MustExec("analyze table t")

	is := executor.GetInfoSchema(tk.Se.(sessionctx.Context))
	table, err := is.TableByName(model.NewCIStr("test"), model.NewCIStr("t"))
	c.Assert(err, IsNil)
	pi := table.Meta().GetPartitionInfo()
	c.Assert(pi, NotNil)
	ids := make([]string, 0, len(pi.Definitions))
	for _, def := range pi.Definitions {
		ids = append(ids, fmt.Sprintf("%d", def.ID))
	}
	result := tk.MustQuery(fmt.Sprintf("select count(distinct(table_id)) from mysql.stats_meta where table_id in (%s)", strings.Join(ids, ",")))
	result.Check(testkit.Rows(fmt.Sprintf("%d", len(ids))))
	result = tk.MustQuery(fmt.Sprintf("select count(distinct(table_id)) from mysql.stats_histograms where table_id in (%s)", strings.Join(ids, ",")))
	result.Check(testkit.Rows(fmt.Sprintf("%d", len(ids))))
	result = tk.MustQuery(fmt.Sprintf("select count(distinct(table_id)) from mysql.stats_buckets where table_id in (%s)", strings.Join(ids, ",")))
	result.Check(testkit.Rows(fmt.Sprintf("%d", len(ids))))
}
