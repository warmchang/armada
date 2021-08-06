package create

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	armadactlCmd "github.com/G-Research/armada/cmd/armadactl/cmd"
	"github.com/G-Research/armada/pkg/api"
	"github.com/G-Research/armada/pkg/client"
)

func init() {
	createCmd.AddCommand(createQueueCmd)
	createQueueCmd.Flags().Float64(
		"priorityFactor", 1,
		"Set queue priority factor - lower number makes queue more important, must be > 0.")
	createQueueCmd.Flags().StringSlice(
		"owners", []string{},
		"Comma separated list of queue owners, defaults to current user.")
	createQueueCmd.Flags().StringSlice(
		"groupOwners", []string{},
		"Comma separated list of queue group owners, defaults to empty list.")
	createQueueCmd.Flags().StringToString(
		"resourceLimits", map[string]string{},
		"Command separated list of resource limits pairs, defaults to empty list. Example: --resourceLimits cpu=0.3,memory=0.2")

}

var createQueueCmd = &cobra.Command{
	Use:   "queue name",
	Short: "Create new queue",
	Long: `Every job submitted to armada needs to be associated with a queue.
Job priority is evaluated within the queue, each queue has its own priority.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := createQueue(cmd, args)
		if err != nil {
			armadactlCmd.ExitWithError(err)
		}
		log.Infof("Queue created")
	},
}

func createQueue(cmd *cobra.Command, args []string) error {
	if _, err := cmd.Flags().GetString("filename"); err == nil {
		return createFromFile(cmd, args, map[string]func(*grpc.ClientConn, []byte) error{"queue": parseAndSendJsonQueueRequest})
	} else {
		return createQueueFromFlags(cmd, args)
	}
}

func createQueueFromFlags(cmd *cobra.Command, args []string) error {
	queueName := args[0]
	priority, _ := cmd.Flags().GetFloat64("priorityFactor")
	owners, _ := cmd.Flags().GetStringSlice("owners")
	groups, _ := cmd.Flags().GetStringSlice("groupOwners")
	resourceLimits, _ := cmd.Flags().GetStringToString("resourceLimits")
	resourceLimitsFloat, err := armadactlCmd.ConvertResourceLimitsToFloat64(resourceLimits)
	if err != nil {
		return err
	}
	queue := api.Queue{
		Name:           queueName,
		PriorityFactor: priority,
		UserOwners:     owners,
		GroupOwners:    groups,
		ResourceLimits: resourceLimitsFloat}

	apiConnectionDetails := client.ExtractCommandlineArmadaApiConnectionDetails()

	return client.WithConnection(apiConnectionDetails, func(conn *grpc.ClientConn) error {
		return sendCreateQueueRequest(conn, queue)
	})
}

func parseAndSendJsonQueueRequest(conn *grpc.ClientConn, queueJsonBytes []byte) error {

	var queue api.Queue
	err := json.Unmarshal(queueJsonBytes, &queue)
	if err != nil {
		return err
	}

	return sendCreateQueueRequest(conn, queue)
}

func sendCreateQueueRequest(conn *grpc.ClientConn, queue api.Queue) error {

	submissionClient := api.NewSubmitClient(conn)
	return client.CreateQueue(submissionClient, &queue)
}
