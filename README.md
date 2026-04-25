# Gossip Glomers challenges in Go

## Fly.io Maelstrom Distributed Systems Challenges — Concept Map

| # | Challenge | Key Concepts |
|---|-----------|-------------|
| 1 | Echo | Maelstrom protocol, stdio IPC, JSON serialization, request/response pattern |
| 2 | Unique ID Generation | Coordination-free design, total availability, uniqueness without consensus, CAP (AP) |
| 3a | Single-Node Broadcast | In-memory state, set deduplication, multi-handler registration |
| 3b | Multi-Node Broadcast | Gossip protocols, topology-aware routing, Send vs RPC, concurrent state with mutexes |
| 3c | Fault-Tolerant Broadcast | Network partitions, retry logic, idempotency, eventual consistency |
| 3d | Efficient Broadcast I | Topology design (tree/grid/mesh), latency-aware routing, batching, gossip tick tuning, performance metrics (msgs/op, latency percentiles) |
| 3e | Efficient Broadcast II | Aggressive optimization, message coalescing, star/flat-tree topologies, latency vs message count tradeoff |
| 4 | Grow-Only Counter | CRDTs (G-Counter), merge functions, join-semilattice, seq-kv store, stateless node design |
| 5a | Single-Node Kafka Log | Append-only logs, offset management, consumer commits, polling, log as a distributed primitive |
| 5b | Multi-Node Kafka Log | Log replication, lin-kv store, compare-and-swap (CAS), CAS retry, per-key routing vs replication |
| 6a | Single-Node Transactions | Transaction processing, micro-operations (r/w), key/value store, read-uncommitted baseline |
| 6b | Read Uncommitted Txns | Write replication, total availability, G0 (dirty writes), consistency as a spectrum |
| 6c | Read Committed Txns | G1a/G1b/G1c anomalies, transaction isolation, abort/conflict handling, stronger consistency under total availability |