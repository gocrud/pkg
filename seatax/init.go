package seatax

import "seata.apache.org/seata-go/v2/pkg/client"

func InitPath(path string) {
	client.InitPath(path)
}

func Init() {
	client.Init()
}
