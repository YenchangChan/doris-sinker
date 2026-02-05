package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Kafka消费指标
	KafkaMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total number of messages consumed from Kafka",
		},
		[]string{"topic"},
	)

	KafkaBytesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_bytes_consumed_total",
			Help: "Total bytes consumed from Kafka",
		},
		[]string{"topic"},
	)

	KafkaConsumeErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consume_errors_total",
			Help: "Total number of Kafka consume errors",
		},
		[]string{"topic", "error_type"},
	)

	// 批次处理指标
	BatchFlushTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "batch_flush_total",
			Help: "Total number of batch flushes",
		},
		[]string{"status"},
	)

	BatchSizeRows = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "batch_size_rows",
			Help:    "Batch size in rows",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
		},
	)

	BatchSizeBytes = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "batch_size_bytes",
			Help:    "Batch size in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600},
		},
	)

	BatchFlushDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "batch_flush_duration_seconds",
			Help:    "Batch flush duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	BatchCurrentRows = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "batch_current_rows",
			Help: "Current number of rows in batch",
		},
	)

	BatchCurrentBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "batch_current_bytes",
			Help: "Current batch size in bytes",
		},
	)

	// Doris写入指标
	DorisStreamLoadTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "doris_stream_load_total",
			Help: "Total number of stream load requests",
		},
		[]string{"status"},
	)

	DorisRowsLoaded = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doris_rows_loaded_total",
			Help: "Total number of rows loaded to Doris",
		},
	)

	DorisRowsFiltered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doris_rows_filtered_total",
			Help: "Total number of rows filtered by Doris",
		},
	)

	DorisStreamLoadDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "doris_stream_load_duration_seconds",
			Help:    "Stream load duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
	)

	DorisStreamLoadRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "doris_stream_load_retries_total",
			Help: "Total number of stream load retries",
		},
	)

	DorisStreamLoadErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "doris_stream_load_errors_total",
			Help: "Total number of stream load errors",
		},
		[]string{"error_type"},
	)

	// JSON解析指标
	JSONParseErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "json_parse_errors_total",
			Help: "Total number of JSON parse errors",
		},
		[]string{"field"},
	)

	FieldMissing = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "field_missing_total",
			Help: "Total number of missing fields",
		},
		[]string{"field"},
	)

	FieldTypeConversionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "field_type_conversion_errors_total",
			Help: "Total number of field type conversion errors",
		},
		[]string{"field", "from_type", "to_type"},
	)
)
