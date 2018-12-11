//
package main

/*

 */

import (
	"818fall18/sylan/p2/dfs"
	"fmt"
	. "github.com/mattn/go-getopt"
	"os"
	"runtime"
	"strconv"
)

//=============================================================================

func main() {
	var c int
	tm := ""
	debug := false
	compress := false
	readOnly := false
	mount := "dss"
	storePath := "db"
	ldbCache := 8 * 1048576
	newfs := ""

	for {
		if c = Getopt("cC:dnm:rs:t:"); c == EOF {
			break
		}

		switch c {
		case 'c':
			compress = !compress // ignore
		case 'C':
			ldbCache, _ = strconv.Atoi(OptArg)
		case 'd':
			debug = !debug
		case 'm':
			mount = OptArg
		case 'n':
			newfs = "NEWFS "
		case 'r':
			readOnly = !readOnly
		case 's':
			storePath = OptArg
		case 't':
			tm = OptArg
		default:
			println("usage: main.go [-d | -c | -m <mountpt> | -t <timespec>]", c)
			os.Exit(1)
		}
	}
	fmt.Printf("\nStartup up with debug %v, compress %v, mountpt: %q, %sstorePath %q, time %q, mprocs %v\n\n",
		debug, compress, mount, newfs, storePath, tm, runtime.GOMAXPROCS(0))

	dfs.Init(debug, compress, mount, newfs != "", storePath, tm, readOnly, ldbCache)

}
