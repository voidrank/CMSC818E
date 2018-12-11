//
package dfs

import (
	//"crypto/sha1"
	//"encoding/base64"
	//"github.com/syndtr/goleveldb/leveldb"
	//"github.com/syndtr/goleveldb/leveldb/opt"
	"log"
	"os"
	"time"
)

func p_out(s string, args ...interface{}) {
	if !debug {
		return
	}
	log.Printf(s, args...)
}

func p_err(s string, args ...interface{}) {
	log.Printf(s, args...)
}

func p_exit(s string, args ...interface{}) {
	log.Printf(s, args...)
	os.Exit(1)
}

// Parse a local time string in the below format.
func parseLocalTime(tm string) (time.Time, error) {
	loc, _ := time.LoadLocation("America/New_York")
	return time.ParseInLocation("2006-1-2T15:04", tm, loc)
}
