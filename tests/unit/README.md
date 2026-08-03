# Unit tests

Unit tests live next to the code they test (Go convention):

- core/policy/policy_test.go — spectrum gate
- core/sim-ops/simprov/simprov_test.go — SIM provisioner
- core/fairwave-control/internal/{api,config,lifecycle,store}/*_test.go

Run: make test (go test ./...).
