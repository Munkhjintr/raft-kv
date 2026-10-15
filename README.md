# raft-kv



A fault-tolerant, replicated key-value store written in Go. The Raft consensus algorithm is implemented from scratch, with no Raft library.



## Features



- Leader election with randomized timeouts

- Log replication, commit rules and ordered apply (Raft paper, Figure 2)

- Persistence of `currentTerm`, `votedFor` and `log` (gob, fsync, atomic rename)

- Replicated KV service: `Get`, `Put`, `Append`, `Delete`

- Exactly-once semantics for client retries (`ClientID` + `Seq`)

- Reads go through the Raft log, so they are linearizable

- Pluggable transport: in-memory network for tests, TCP `net/rpc` for real nodes

- HTTP API on every node



## Architecture



```mermaid

flowchart LR

&#x20;   C[Client / curl] -->|HTTP| S[kv.Server]

&#x20;   S -->|Propose| R[raft.Raft]

&#x20;   R -->|AppendEntries / RequestVote| T[(Transport)]

&#x20;   T --> P1[Peer node]

&#x20;   T --> P2[Peer node]

&#x20;   R -->|persist| D[(FileStorage)]

&#x20;   R -->|ApplyCh, committed entries| S

&#x20;   S -->|execute| M[kv.Store]

```



| Package | Responsibility |

|---|---|

| `raft` | Election, replication, commit, persistence, transports |

| `kv` | State machine, server, clerk (client), HTTP handler |

| `cmd/raft-kv` | Node binary |

| `cmd/bench` | Benchmark runner |

| `cmd/plot` | SVG chart generator |



## Run a 3-node cluster



Use one terminal per node.



```

go run ./cmd/raft-kv -id 0 -peers "0=127.0.0.1:7000,1=127.0.0.1:7001,2=127.0.0.1:7002" -http 127.0.0.1:8000

go run ./cmd/raft-kv -id 1 -peers "0=127.0.0.1:7000,1=127.0.0.1:7001,2=127.0.0.1:7002" -http 127.0.0.1:8001

go run ./cmd/raft-kv -id 2 -peers "0=127.0.0.1:7000,1=127.0.0.1:7001,2=127.0.0.1:7002" -http 127.0.0.1:8002

```



Find the leader and write to it:



```

curl http://127.0.0.1:8000/status

curl -X PUT -d "hello" http://127.0.0.1:8000/kv/greeting

curl http://127.0.0.1:8000/kv/greeting

```



Followers answer `503 wrong leader`. State is stored in `data/node-N.bin`.



| Method | Path | Meaning |

|---|---|---|

| GET | `/kv/{key}` | read |

| PUT | `/kv/{key}` | set |

| POST | `/kv/{key}` | append |

| DELETE | `/kv/{key}` | delete |

| GET | `/status` | term and leadership |



## Testing



```

go test -race -count=3 ./...

```



Covered scenarios:



- Leader election, stable leadership, re-election after leader failure

- No leader without a quorum

- Log replication, lagging follower catch-up, committed entries survive leader failure

- Persistence: term, vote and log restored; no second vote in the same term after restart; all nodes restart at once

- Exactly-once application of duplicate client requests

- Network partitions: leader in the minority cannot commit, uncommitted entries are discarded after healing, no commit without a quorum, convergence after healing



All tests run with the Go race detector.



## Failure demo (real processes)



1. Start 3 nodes, the leader (node 0) accepts `greeting=hello`.

2. Kill the leader. Node 2 becomes leader in the next term (603 to

