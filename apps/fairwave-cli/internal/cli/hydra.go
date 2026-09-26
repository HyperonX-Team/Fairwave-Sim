package cli

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/bearer"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/fabric"
	"github.com/spf13/cobra"
)

// hydraCmd groups the cooperative bearer multiplexing commands.
func hydraCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hydra",
		Short: "Cooperative bearer multiplexing (weave SIMs, modems, and boxes into one link)",
		Long: "Hydra multiplexes a single subscriber flow across many independent SIMs,\n" +
			"modems, and boxes, reassembling it in order at a weave anchor. One box is\n" +
			"limited by its own modem; a weave is limited by the sum of its threads.",
	}
	cmd.AddCommand(
		hydraStatusCmd(),
		hydraThreadsCmd(),
		hydraThreadAddCmd(),
		hydraThreadRemoveCmd(),
		hydraWeavesCmd(),
		hydraWeaveCreateCmd(),
		hydraWeaveRemoveCmd(),
		hydraWeaveStatsCmd(),
		hydraBenchCmd(),
		hydraStripCmd(),
		hydraIngestCmd(),
	)
	return cmd
}

func hydraStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show weave summary: threads, capacity, frames",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var st fabric.Status
			if err := c.get("/v1/hydra/status", &st); err != nil {
				return err
			}
			fmt.Printf("hydra anchor:     %s\n", st.Anchor)
			fmt.Printf("threads:          %d (%d up)\n", st.Threads, st.ThreadsUp)
			fmt.Printf("weaves:           %d\n", st.Weaves)
			fmt.Printf("aggregate:        %.0f Mbps\n", st.AggregateMbps)
			fmt.Printf("frames sent:      %d\n", st.FramesSent)
			fmt.Printf("payloads delivered: %d\n", st.Delivered)
			return nil
		},
	}
}

func hydraThreadsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "threads",
		Short: "List bearer threads",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var threads []bearer.Thread
			if err := c.get("/v1/hydra/threads", &threads); err != nil {
				return err
			}
			if len(threads) == 0 {
				fmt.Println("no threads")
				return nil
			}
			fmt.Printf("%-16s %-10s %-8s %-10s %-8s %-6s\n", "ID", "BOX", "MBPS", "RTT(ms)", "LOSS(%)", "UP")
			for _, t := range threads {
				fmt.Printf("%-16s %-10s %-8.0f %-10.1f %-8.1f %-6v\n", t.ID, t.Box, t.Mbps, t.RTTms, t.LossPct, t.Up)
			}
			return nil
		},
	}
}

func hydraThreadAddCmd() *cobra.Command {
	var req api.HydraThreadRequest
	cmd := &cobra.Command{
		Use:   "thread-add",
		Short: "Add or update a bearer thread (one SIM/modem/path)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var out bearer.Thread
			if err := c.post("/v1/hydra/threads", req, &out); err != nil {
				return err
			}
			fmt.Printf("thread %s: %.0f Mbps, rtt %.1f ms, loss %.1f%%, up=%v\n",
				out.ID, out.Mbps, out.RTTms, out.LossPct, out.Up)
			return nil
		},
	}
	cmd.Flags().StringVar(&req.ID, "id", "", "thread id (required)")
	cmd.Flags().StringVar(&req.Box, "box", "", "originating box")
	cmd.Flags().StringVar(&req.SIM, "sim", "", "SIM/IMSI backing this thread")
	cmd.Flags().Float64Var(&req.Mbps, "mbps", 0, "measured capacity in Mbps")
	cmd.Flags().Float64Var(&req.RTTms, "rtt", 0, "round-trip time in ms")
	cmd.Flags().Float64Var(&req.LossPct, "loss", 0, "loss percentage")
	cmd.Flags().BoolVar(&req.Up, "up", true, "thread is available for scheduling")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("mbps")
	return cmd
}

func hydraThreadRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "thread-remove <id>",
		Short: "Remove a bearer thread",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			if _, err := c.raw("DELETE", "/v1/hydra/threads/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Printf("thread %s removed\n", args[0])
			return nil
		},
	}
}

func hydraWeavesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "weaves",
		Short: "List weaves",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var weaves []fabric.Weave
			if err := c.get("/v1/hydra/weaves", &weaves); err != nil {
				return err
			}
			if len(weaves) == 0 {
				fmt.Println("no weaves")
				return nil
			}
			fmt.Printf("%-16s %-12s %-12s %s\n", "ID", "ANCHOR", "MODE", "THREADS")
			for _, w := range weaves {
				fmt.Printf("%-16s %-12s %-12s %s\n", w.ID, w.Anchor, w.Mode, strings.Join(w.Threads, ","))
			}
			return nil
		},
	}
}

func hydraWeaveCreateCmd() *cobra.Command {
	var id, anchor, threads string
	cmd := &cobra.Command{
		Use:   "weave-create",
		Short: "Bind threads to an anchor as one logical link",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ids := splitCSV(threads)
			if len(ids) == 0 {
				return fmt.Errorf("--threads must list at least one thread id")
			}
			c := newClient(cmd)
			var w fabric.Weave
			if err := c.post("/v1/hydra/weaves", api.HydraWeaveRequest{ID: id, Anchor: anchor, Threads: ids}, &w); err != nil {
				return err
			}
			fmt.Printf("weave %s -> anchor %s over %d threads: %s\n", w.ID, w.Anchor, len(w.Threads), strings.Join(w.Threads, ","))
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "weave id (required)")
	cmd.Flags().StringVar(&anchor, "anchor", "", "anchor name (defaults to the node's local anchor)")
	cmd.Flags().StringVar(&threads, "threads", "", "comma-separated thread ids (required)")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("threads")
	return cmd
}

func hydraWeaveRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "weave-remove <id>",
		Short: "Delete a weave",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			if _, err := c.raw("DELETE", "/v1/hydra/weaves/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Printf("weave %s removed\n", args[0])
			return nil
		},
	}
}

func hydraWeaveStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "weave-stats <id>",
		Short: "Show a weave's live counters",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var st fabric.WeaveStats
			if err := c.get("/v1/hydra/weaves/"+args[0]+"/stats", &st); err != nil {
				return err
			}
			fmt.Printf("weave %s (anchor %s, mode %s)\n", st.ID, st.Anchor, st.Mode)
			fmt.Printf("  aggregate:   %.0f Mbps\n", st.AggregateMbps)
			fmt.Printf("  frames sent: %d (%d bytes)\n", st.FramesSent, st.BytesSent)
			fmt.Printf("  reorder:     delivered=%d reordered=%d dup=%d too-far=%d depth<=%d\n",
				st.Reorder.Delivered, st.Reorder.Reordered, st.Reorder.Duplicates,
				st.Reorder.TooFar, st.Reorder.MaxDepth)
			fmt.Printf("  bench:       %d run(s), %d payload(s) reassembled\n", st.BenchRuns, st.BenchDelivered)
			return nil
		},
	}
}

func hydraBenchCmd() *cobra.Command {
	var packets, pktBytes int
	cmd := &cobra.Command{
		Use:   "bench <weave-id>",
		Short: "Synthetic stripe/reassemble bench proving the aggregate speedup",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var res fabric.BenchResult
			if err := c.post("/v1/hydra/weaves/"+args[0]+"/bench",
				api.HydraBenchRequest{Packets: packets, PktBytes: pktBytes}, &res); err != nil {
				return err
			}
			fmt.Printf("hydra bench: %s\n", res.WeaveID)
			fmt.Printf("  packets:        %d (%d bytes)\n", res.Packets, res.Bytes)
			fmt.Printf("  delivered:      %d\n", res.Delivered)
			fmt.Printf("  single thread:  %.0f Mbps\n", res.SingleThreadMbps)
			fmt.Printf("  aggregate:      %.0f Mbps\n", res.AggregateMbps)
			fmt.Printf("  speedup:        %.2fx\n", res.Speedup)
			fmt.Printf("  reorder events: %d (max depth %d)\n", res.Reorder.Reordered, res.Reorder.MaxDepth)
			return nil
		},
	}
	cmd.Flags().IntVar(&packets, "packets", 300, "number of packets to stripe")
	cmd.Flags().IntVar(&pktBytes, "pkt-bytes", 1200, "payload size in bytes")
	return cmd
}

func hydraStripCmd() *cobra.Command {
	var payload string
	cmd := &cobra.Command{
		Use:   "strip <weave-id>",
		Short: "Lab: assign one payload to a thread and print the framed packet",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var res api.HydraStripResponse
			if err := c.post("/v1/hydra/weaves/"+args[0]+"/strip",
				api.HydraStripRequest{PayloadB64: base64.StdEncoding.EncodeToString([]byte(payload))}, &res); err != nil {
				return err
			}
			fmt.Printf("seq %d -> thread %s (%d bytes)\n", res.Seq, res.ThreadID, res.FrameLen)
			fmt.Println(res.FrameB64)
			return nil
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "hello-hydra", "payload text to stripe")
	return cmd
}

func hydraIngestCmd() *cobra.Command {
	var frame string
	cmd := &cobra.Command{
		Use:   "ingest <weave-id>",
		Short: "Lab: feed a framed packet back into reassembly",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var res api.HydraIngestResponse
			if err := c.post("/v1/hydra/weaves/"+args[0]+"/ingest",
				api.HydraIngestRequest{FrameB64: frame}, &res); err != nil {
				return err
			}
			fmt.Printf("delivered %d payload(s)\n", res.Delivered)
			return nil
		},
	}
	cmd.Flags().StringVar(&frame, "frame", "", "base64 frame from `hydra strip` (required)")
	_ = cmd.MarkFlagRequired("frame")
	return cmd
}

// splitCSV trims and drops empty fields.
func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
