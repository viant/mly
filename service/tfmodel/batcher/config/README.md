# Unbounded configurations and behavior

```
MBS = MaxBatchSize
BW  = BatchWait
MEC = MaxEvaluatorConcurrency
MQB = MaxQueuedBatches

N is N as in number N.

MBS BW  MEC MQB - unbounded behavior
0   0   0   0   = invalid configuration - use defaults (add bound BatchWait)
N   0   0   0   = infinite running evaluations with batch size MaxBatchSize
0   N   0   0   = infinite running evaluations with variable batch size 
N   N   0   0   = infinite running evaluations with batch size MaxBatchSize or less
0   0   N   0   = invalid configuration - use defaults (bound BatchWait)
N   0   N   0   = infinite queued batches of size MaxBatchSize
0   N   N   0   = infinite queued batches with variable batch size
N   N   N   0   = infinite queued batches with batch size MaxBatchSize or less
0   0   0   N   = invalid configuration - use defaults (add bound BatchWait)
N   0   0   N   = infinite running evaluations with batch size MaxBatchSize
0   N   0   N   = infinite running evaluations with variable batch size 
N   N   0   N   = infinite running evaluations with batch size MaxBatchSize or less
0   0   N   N   = invalid configuration - use defaults (add bound BatchWait)
N   0   N   N   = shed with N queued batches of size MaxBatchSize
0   N   N   N   = infinite size batch with N queued batches of variable batch size
N   N   N   N   = shed with N queued batches of size MaxBatchSize or less
```
