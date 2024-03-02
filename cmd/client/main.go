package main

import (
	"context"
	"fmt"
	"net/http"

	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/twitchtv/twirp"
)

func main() {
	fmt.Println("Setting up client")

	client := protos.NewPodProtobufClient("http://localhost:8081", http.DefaultClient, twirp.WithClientPathPrefix("rpc/podcast"))

	response, err := client.GetPodcast(context.Background(), &protos.GetPodReq{
		Id: "ad6dfab0-3454-4f3b-bca0-37485b1a9b3c",
	})
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println("response:", response)
}
