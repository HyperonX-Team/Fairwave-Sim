package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

// doctorCmd runs local environment diagnostics (no network needed).
func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose this machine for Fairwave deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			fail := 0
			check := func(name string, ok bool, note string) {
				mark := "OK "
				if !ok {
					mark = "FAIL"
					fail++
				}
				fmt.Printf("[%s] %-28s %s\n", mark, name, note)
			}

			fmt.Printf("fairwave doctor (%s %s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

			// kernel
			uname, err := exec.LookPath("uname")
			if err == nil {
				out, _ := exec.Command(uname, "-r").Output()
				check("kernel", len(out) > 0, "uname -r")
			} else {
				check("kernel", false, "uname not found (POSIX env expected)")
			}

			// docker
			if p, err := exec.LookPath("docker"); err == nil {
				check("docker", p != "", p)
			} else {
				check("docker", false, "docker not on PATH (required for make lab-up)")
			}

			// make
			if _, err := exec.LookPath("make"); err != nil {
				check("make", false, "GNU make not on PATH")
			} else {
				check("make", true, "")
			}

			// go
			if p, err := exec.LookPath("go"); err == nil {
				check("go", p != "", p)
			} else {
				check("go", false, "go not on PATH (only needed to build from source)")
			}

			// control plane reachability
			c := newClient(cmd)
			var st map[string]interface{}
			if err := c.get("/v1/healthz", &st); err == nil {
				check("control plane", true, c.base)
			} else {
				check("control plane", false, fmt.Sprintf("unreachable at %s", c.base))
			}

			// rf mode safety
			mode := os.Getenv("FAIRWAVE_SERVER_MODE")
			if mode == "rf" {
				check("rf mode", false, "FAIRWAVE_SERVER_MODE=rf is set; lab mode is the safe default")
			} else {
				check("rf mode", true, "default lab/no-RF")
			}

			if fail > 0 {
				fmt.Printf("\n%d issue(s) found — see docs/ops/troubleshooting.md\n", fail)
				return nil
			}
			fmt.Println("\nall checks passed")
			return nil
		},
	}
}
