package dfs

/*
import (
	// "launchpad.net/goamz/aws"
	// "launchpad.net/goamz/s3"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	//	"github.com/mitchellh/goamz/aws"
	//	"github.com/mitchellh/goamz/s3"
)

var bucket *s3.Bucket

func init() {
	auth := aws.Auth{
		AccessKey: "AKIAJNOR2KM4FGREDAQA",
		SecretKey: "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",
	}

	connection := s3.New(auth, aws.USEast)
	bucket = connection.Bucket("motefs2")
}

func s3Put(key string, data []byte) error {
	p_out("s3 put %q\n", key)
	if err := bucket.Put(key, data, "Content-MD", ""); err != nil {
		p_err("bkt ERROR %v\n", err)
		return err
	} else {
		return nil
	}
}

func s3Get(key string) []byte {
	p_out("s3 get %q\n", key)
	data, err := bucket.Get(key)
	if err != nil {
		p_err("bkt get %q ERROR %v\n", key, err)
		return nil
	} else {
		return data
	}
}
*/
