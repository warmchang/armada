package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/armadaproject/armada/internal/armadactl"
)

func getSchedulingReportCmd(a *armadactl.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "scheduling-report",
		Short:        "Get scheduler reports",
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initParams(cmd, a.Params)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbosity, err := cmd.Flags().GetCount("verbose")
			if err != nil {
				return err
			}

			queueName, err := cmd.Flags().GetString("queue")
			if err != nil {
				return err
			}
			queueName = strings.TrimSpace(queueName)
			if queueName != "" {
				fmt.Print("This flag is deprecated and will be removed in a future release. Please use `queue-report` instead.\n")
				return a.GetSchedulingReportForQueue(queueName, int32(verbosity))
			}

			jobId, err := cmd.Flags().GetString("job-id")
			if err != nil {
				return err
			}
			jobId = strings.TrimSpace(jobId)
			if jobId != "" {
				fmt.Print("This flag is deprecated and will be removed in a future release. Please use `job-report` instead.\n")
				return a.GetSchedulingReportForJob(jobId, int32(verbosity))
			}

			return a.GetSchedulingReport(int32(verbosity))
		},
	}

	cmd.Flags().CountP("verbose", "v", "report verbosity; repeat (e.g., -vvv) to increase verbosity")

	cmd.Flags().String("queue", "", "get scheduler reports relevant for this queue; mutually exclusive with --job-id")
	cmd.Flags().String("job-id", "", "get scheduler reports relevant for this job; mutually exclusive with --queue")
	cmd.MarkFlagsMutuallyExclusive("queue", "job-id")

	return cmd
}

func getQueueSchedulingReportCmd(a *armadactl.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue-report <queue-name>",
		Short: "Get queue scheduler reports",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initParams(cmd, a.Params)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbosity, err := cmd.Flags().GetCount("verbose")
			if err != nil {
				return err
			}

			queueName := args[0]
			queueName = strings.TrimSpace(queueName)

			return a.GetQueueSchedulingReport(queueName, int32(verbosity))
		},
	}

	cmd.Flags().CountP("verbose", "v", "report verbosity; repeat (e.g., -vvv) to increase verbosity")

	return cmd
}

func getJobSchedulingReportCmd(a *armadactl.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job-report <job-id>",
		Short: "Get job scheduler reports",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initParams(cmd, a.Params)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			jobId := args[0]
			jobId = strings.TrimSpace(jobId)
			return a.GetJobSchedulingReport(jobId)
		},
	}
	return cmd
}
