# Geo-Distribution Tradeoff Results

| Deployment | Successful rounds | Avg write | P95 write | Avg read | Avg convergence | Immediate stale rate |
|---|---:|---:|---:|---:|---:|---:|
| Co-located | 100 | 266.26 ms | 684.80 ms | 251.60 ms | 799.20 ms | 0.00% |
| Geo-distributed | 100 | 311.34 ms | 686.90 ms | 363.05 ms | 1090.10 ms | 0.00% |

## Comparison

- Geo-distributed avg write latency ratio: 1.17x
- Geo-distributed avg read latency ratio: 1.44x
- Geo-distributed avg convergence ratio: 1.36x

## Interpretation Guide

- Higher write latency in the geo-distributed setup indicates the cost of cross-region Raft consensus.
- Lower read latency on nearby replicas would indicate the benefit of client proximity.
- A higher immediate stale rate means followers are more likely to lag right after a write.
- Higher convergence time means replicas take longer to observe the latest committed value.
