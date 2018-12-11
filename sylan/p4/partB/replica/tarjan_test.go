package main

import (
	"818fall18/sylan/p4/partB/pb" /* MODIFY ME */
	"fmt"
	"strings"
	"testing"
)

type Inst struct {
	rep  int
	seq  int64
	deps []int64
}

//=====================================================================

func doTest(nm string, insts []*Inst, t *testing.T) {
	N = 5
	debug = 0
	executed = []int64{-1, -1, -1, -1, -1}
	logs = make([][]*pb.Instance, N)
	for i := range logs {
		logs[i] = make([]*pb.Instance, 0, LOG_MAX)
		executed[i] = -1
	}

	for _, inst := range insts {
		r := int64(inst.rep)
		l := int64(len(logs[r]))
		logs[r] = append(logs[r], &pb.Instance{Ops: []*pb.Operation{&pb.Operation{Key: "text", Value: []byte(fmt.Sprintf("%d", l))}},
			Rep: r, Slot: l, Seq: inst.seq, State: pb.InstState_COMMITTED, Deps: inst.deps})
	}

	sscs := executeSome()
	ss := []string{}
	for _, scc := range sscs {
		ss = append(ss, "scc {")
		for _, v := range scc {
			ss = append(ss, fmt.Sprintf("\t%v", v))
		}
		ss = append(ss, "}")
	}
	p_err("\n%s: \n%v\n\n", nm, strings.Join(ss, "\n"))
}

//=====================================================================

// two SCCs

func TestTarjan(t *testing.T) {

	var insts = []*Inst{
		&Inst{0, 0, []int64{-1, -1, -1, -1, 0}},
		&Inst{0, 1, []int64{-1, -1, -1, -1, 1}},
		&Inst{4, 0, []int64{0, -1, -1, -1, -1}},
		&Inst{4, 1, []int64{1, -1, -1, -1, -1}},
	}
	doTest("one", insts, t)
	if (executed[0] != 1) || (executed[4] != 1) {
		t.Errorf("Bad executions: %v\n", executed)
	}
}

func TestTarjan2(t *testing.T) {
	// two SCCs   0 -> 2 -> 1 -> 0,  3 -> 0  (not part of anything)
	var insts2 = []*Inst{
		&Inst{0, 0, []int64{-1, -1, 0, -1, -1}},
		&Inst{1, 0, []int64{0, -1, -1, -1, -1}},
		&Inst{2, 0, []int64{-1, 0, -1, -1, -1}},
		&Inst{3, 0, []int64{0, -1, -1, -1, -1}},
	}
	doTest("two", insts2, t)
	if (executed[0] != 0) || (executed[1] != 0) || (executed[2] != 0) || (executed[3] != 0) || (executed[4] != -1) {
		t.Errorf("Bad executions: %v\n", executed)
	}
}

func TestTarjan3(t *testing.T) {
	// two SCCs   0->1, 2->0, 2->1, 4->3, 5->0, 5->4    2->1,4->2,5->3
	var insts3 = []*Inst{
		&Inst{0, 1, []int64{-1, 0, -1, -1, -1}},
		&Inst{1, 0, []int64{-1, -1, -1, -1, -1}},
		&Inst{1, 2, []int64{0, 0, -1, -1, -1}},
		&Inst{2, 0, []int64{-1, -1, -1, -1, -1}},
		&Inst{1, 3, []int64{-1, 1, 0, -1, -1}},
		&Inst{2, 4, []int64{0, 2, 0, -1, -1}},
	}
	doTest("three", insts3, t)
	if (executed[0] != 0) || (executed[1] != 2) || (executed[2] != 1) || (executed[3] != -1) || (executed[4] != -1) {
		t.Errorf("Bad executions: %v\n", executed)
	}

}
