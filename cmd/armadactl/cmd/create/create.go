package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/G-Research/armada/pkg/client"

	armadactlCmd "github.com/G-Research/armada/cmd/armadactl/cmd"
)

func InitCreate() {

	createCmd.Flags().StringP(
		"filename", "f", "",
		"Json file to create queue from")

	armadactlCmd.AddTopLevelCommand(createCmd)
}

var kindToCreator = map[string]func(*grpc.ClientConn, []byte) error{
	"queue": parseAndSendJsonQueueRequest,
}

var createCmd = &cobra.Command{
	Use:   "create -f",
	Short: "Create new resources",
	Long:  `Create new resources (for now only queues) from a file or standard input`,

	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := cmd.Flags().GetString("filename"); err == nil {
			armadactlCmd.ErrorCheck(createFromFile(cmd, args, kindToCreator))

		} else {
			armadactlCmd.ExitWithError(errors.New("You need to specify what to create, either by using the -f option to create from a file, or by specifying a kind and name (e.g. create queue my-queue)."))
		}
	},
}

func createFromFile(cmd *cobra.Command, args []string, kindToCreator map[string]func(*grpc.ClientConn, []byte) error) error {
	filename, err := cmd.Flags().GetString("filename")
	if err != nil {
		return errors.New("You need to specify -f or --filename on the command line")
	}

	jsonBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Error reading file %s: %v", filename, err)
	}

	kind := getKind(jsonBytes)

	creator, exists := kindToCreator[kind]
	if !exists {
		return fmt.Errorf("Unknown kind %s in file %s", kind, filename)
	}

	apiConnectionDetails := client.ExtractCommandlineArmadaApiConnectionDetails()

	return client.WithConnection(apiConnectionDetails, func(conn *grpc.ClientConn) error {
		return creator(conn, jsonBytes)
	})
}

func getKind(jsonBytes []byte) string {
	var kindHolder struct{ Kind string }
	json.Unmarshal(jsonBytes, &kindHolder)
	return kindHolder.Kind
}
