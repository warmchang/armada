package client

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func WithConnection(apiConnectionDetails *ApiConnectionDetails, action func(*grpc.ClientConn) error) error {
	conn, err := CreateApiConnection(apiConnectionDetails)

	if err != nil {
		log.Errorf("Failed to connect to api because %s", err)
		return err
	}
	defer conn.Close()

	return action(conn)
}
