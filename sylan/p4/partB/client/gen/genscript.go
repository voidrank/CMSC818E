package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	. "github.com/mattn/go-getopt"
)

const (
	ROUND_ROBIN = iota
	RANDOM      = iota
	FIXED       = iota
)

const (
	WRITES100 = iota
	WRITES90  = iota
	WRITES10  = iota
	WRITES0   = iota
	WRITEX100 = iota
	WRITEX90  = iota
	WRITEX10  = iota
	WRITEX0   = iota
)

type ScriptAccess struct {
	R int
	T int
	K string
	V string
}

type Script struct {
	Accesses []ScriptAccess
}

var (
	scriptFile      = "script.json"
	N               = 3
	num             = 100
	conflictPercent = 1.0
	dist            = ROUND_ROBIN
	accessTypes     = WRITES100
)

//=====================================================================

func main() {
	for {
		if c := Getopt("n:N:c:s:rf"); c == EOF {
			break
		} else {
			switch c {
			case 'n':
				num, _ = strconv.Atoi(OptArg)
			case 'N':
				N, _ = strconv.Atoi(OptArg)
			case 'c':
				conflictPercent, _ = strconv.ParseFloat(OptArg, 64)
			case 's':
				scriptFile = OptArg
			case 'r':
				dist = RANDOM
			case 'f':
				dist = FIXED
			case 't':
				accessTypes, _ = strconv.Atoi(OptArg)
			}
		}
	}

	//=====================================================================
	// Generate our script
	script := new(Script)
	script.Accesses = make([]ScriptAccess, num)

	for i := 0; i < num; i++ {
		rep := i % N
		script.Accesses[i] = ScriptAccess{rep, 0, "key", "val"}
	}

	b, _ := json.Marshal(script)

	nm := fmt.Sprintf("script_%v_%v_%v_%v_%v.json", num, N, dist, int(100*conflictPercent), accessTypes)
	ioutil.WriteFile(nm, b, 0644)
}
