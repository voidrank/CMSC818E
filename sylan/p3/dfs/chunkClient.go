package dfs

import (
	pbc "818fall18/sylan/p3/pbc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"time"
)

var ()

const (
	address = "localhost:50051"
)

var (
	client pbc.ChunkClient
)

func Init() {
	conn, _ := grpc.Dial(address, grpc.WithInsecure())
	client = pbc.NewChunkClient(conn)
}

func ChunkPut(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.Put(ctx, &pbc.ChunkPutRequest{Key: key, Value: value})
	if err != nil {
		return err
	} else {
		return nil
	}
}

func ChunkGet(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := client.Get(ctx, &pbc.ChunkGetRequest{Key: key})
	if err != nil {
		return []byte(""), err
	} else {
		return r.Value, nil
	}
}
