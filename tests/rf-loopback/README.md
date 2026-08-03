# RF loopback (dry-run)

make rf-dry-run validates RF configuration paths **without transmitting**:
- deploy/docker-compose.rf.yml must refuse to start unless the TX gate is satisfied
- frequency plan consistency is checked by config validation

No radio device is ever opened by these checks.
