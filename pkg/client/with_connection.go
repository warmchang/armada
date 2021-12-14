package client

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func WithConnection(apiConnectionDetails *ApiConnectionDetails, action func(*grpc.ClientConn)) {
	conn, err := CreateApiConnection(apiConnectionDetails)

	if err != nil {
		log.Errorf("Error connecting to API: %q", err)
		return
	}
	defer conn.Close()

	action(conn)
}
