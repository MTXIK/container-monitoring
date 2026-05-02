CREATE TABLE IF NOT EXISTS container_metrics
(
    collected_at DateTime64(3),
    node_id String,
    container_id String,
    name String,
    cpu_percent Float64,
    memory_bytes UInt64,
    rx_bytes UInt64,
    tx_bytes UInt64,
    block_read UInt64,
    block_write UInt64
)
ENGINE = MergeTree
PARTITION BY toDate(collected_at)
ORDER BY (node_id, container_id, collected_at);

CREATE TABLE IF NOT EXISTS container_events
(
    occurred_at DateTime64(3),
    node_id String,
    container_id String,
    name String,
    type String
)
ENGINE = MergeTree
PARTITION BY toDate(occurred_at)
ORDER BY (node_id, container_id, occurred_at);
