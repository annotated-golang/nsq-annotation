package nsqd

import (
	"crypto/md5"
	"crypto/tls"
	"hash/crc32"
	"io"
	"log"
	"os"
	"time"

	"github.com/nsqio/nsq/internal/lg"
)

type Options struct {
	// basic options （普通配置）
	ID        int64       `flag:"node-id" cfg:"id"` // nsqd id
	LogLevel  lg.LogLevel `flag:"log-level"`        // 日志打印等级
	LogPrefix string      `flag:"log-prefix"`       // 打印日志内容每行数据的前缀，用来识别logger
	Logger    Logger      // Logger 对象

	TCPAddress               string        `flag:"tcp-address"`                                                   // TCP 监听地址
	HTTPAddress              string        `flag:"http-address"`                                                  // HTTP 监听地址
	HTTPSAddress             string        `flag:"https-address"`                                                 // HTTPS 监听地址
	BroadcastAddress         string        `flag:"broadcast-address"`                                             // 注册到lookup的地址，默认为"yourhost.local"
	BroadcastTCPPort         int           `flag:"broadcast-tcp-port"`                                            // 注册到lookup的tcp端口，默认为nsqd监听的端口
	BroadcastHTTPPort        int           `flag:"broadcast-http-port"`                                           // 注册到lookup的http端口，默认为nsqd监听的端口
	NSQLookupdTCPAddresses   []string      `flag:"lookupd-tcp-address" cfg:"nsqlookupd_tcp_addresses"`            // lookup的tcp地址
	AuthHTTPAddresses        []string      `flag:"auth-http-address" cfg:"auth_http_addresses"`                   // auth 服务器地址
	HTTPClientConnectTimeout time.Duration `flag:"http-client-connect-timeout" cfg:"http_client_connect_timeout"` // http连接超时默认2s
	HTTPClientRequestTimeout time.Duration `flag:"http-client-request-timeout" cfg:"http_client_request_timeout"` // http请求超时默认2s

	// diskqueue options (硬盘队列配置)
	DataPath        string        `flag:"data-path"`          // 数据地址来保存硬盘备份消息
	MemQueueSize    int64         `flag:"mem-queue-size"`     // 每个topic下的channel，在内存中保存消息的数量，默认为10000
	MaxBytesPerFile int64         `flag:"max-bytes-per-file"` // 在硬盘队列文件滚动前，文件的最大字节数。默认为 104857600
	SyncEvery       int64         `flag:"sync-every"`         // 多少个消息进行，fsync把消息刷到硬盘, 默认为2500
	SyncTimeout     time.Duration `flag:"sync-timeout"`       // 多久触发一次fsync, 默认为2s
	// Sync 这两个参数来看可能有消息丢失的问题？nsqd挂掉但是fsync没触发？

	QueueScanInterval        time.Duration // 队列检查区间，进行检查部分channel是否变脏，满足一定阈值则持续的去清理。如果有了解redis的过期算法，这个很好理解。（probabilistic expiration algorithm）
	QueueScanRefreshInterval time.Duration // 对队列的的大小进行resize
	QueueScanSelectionCount  int           `flag:"queue-scan-selection-count"` // 清理worker的队列大小
	QueueScanWorkerPoolMax   int           `flag:"queue-scan-worker-pool-max"` // Worker吃最大数量
	QueueScanDirtyPercent    float64       // 脏阈值百分比，如果超过这个百分比则进行操作。

	// msg and command options （消息和命令配置）
	MsgTimeout    time.Duration `flag:"msg-timeout"`     // 消息过期时间，在重新投递到队列的等待的时长
	MaxMsgTimeout time.Duration `flag:"max-msg-timeout"` // 消息的最大超时时间，默认15m0s
	MaxMsgSize    int64         `flag:"max-msg-size"`    // 消息的最大比特数
	MaxBodySize   int64         `flag:"max-body-size"`   // 消息命令内容的最大比特数
	MaxReqTimeout time.Duration `flag:"max-req-timeout"` // 最大重新投递的超时时间
	ClientTimeout time.Duration //客户端超时时间

	// client overridable configuration options （客户端可重写配置）
	MaxHeartbeatInterval   time.Duration `flag:"max-heartbeat-interval"`    // 最大客户端配置的心跳时间, 默认15m
	MaxRdyCount            int64         `flag:"max-rdy-count"`             // 客户端最大准备接受数量，用来控制nsqd不发送过多的消息给客户端。默认2500
	MaxOutputBufferSize    int64         `flag:"max-output-buffer-size"`    // 客户端配置的客户端输出buffer的最大比特数，默认为65536
	MaxOutputBufferTimeout time.Duration `flag:"max-output-buffer-timeout"` // 客户端输出，flush到客户端的最大时间间隔，默认30s
	MinOutputBufferTimeout time.Duration `flag:"min-output-buffer-timeout"` // 客户端输出，flush到客户端的最小时间间隔，默认25ms
	OutputBufferTimeout    time.Duration `flag:"output-buffer-timeout"`     // 客户端输出，flush到客户端的时间间隔，默认250ms
	MaxChannelConsumers    int           `flag:"max-channel-consumers"`     // 每个nsqd实例的最大channel消费者连接数

	// statsd integration （统计数据配置）
	StatsdAddress       string        `flag:"statsd-address"`         // UDP地址统计数据daemon，推送统计数据
	StatsdPrefix        string        `flag:"statsd-prefix"`          // 推送数据的key的前缀，默认为"nsq.%s"
	StatsdInterval      time.Duration `flag:"statsd-interval"`        // 推送数据频率
	StatsdMemStats      bool          `flag:"statsd-mem-stats"`       // 记录内存和GC相关数据
	StatsdUDPPacketSize int           `flag:"statsd-udp-packet-size"` // 数据UDP包大小

	// e2e message latency （消息时延）
	E2EProcessingLatencyWindowTime  time.Duration `flag:"e2e-processing-latency-window-time"`                                         // 处理时延窗口时间
	E2EProcessingLatencyPercentiles []float64     `flag:"e2e-processing-latency-percentile" cfg:"e2e_processing_latency_percentiles"` // 处理时延时间分布

	// TLS config （TLS配置）
	TLSCert             string `flag:"tls-cert"`
	TLSKey              string `flag:"tls-key"`
	TLSClientAuthPolicy string `flag:"tls-client-auth-policy"`
	TLSRootCAFile       string `flag:"tls-root-ca-file"`
	TLSRequired         int    `flag:"tls-required"`
	TLSMinVersion       uint16 `flag:"tls-min-version"`

	// compression （压缩）
	DeflateEnabled  bool `flag:"deflate"`           // 开启客户端压缩
	MaxDeflateLevel int  `flag:"max-deflate-level"` // 最大压缩等级
	SnappyEnabled   bool `flag:"snappy"`            // 开启snappy协商feature
}

func NewOptions() *Options {
	// 获取os主机名
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	// 根据主机名来生成Id
	h := md5.New()
	io.WriteString(h, hostname)
	defaultID := int64(crc32.ChecksumIEEE(h.Sum(nil)) % 1024)

	// 生成默认配置
	return &Options{
		ID:        defaultID,
		LogPrefix: "[nsqd] ",
		LogLevel:  lg.INFO,

		TCPAddress:        "0.0.0.0:4150",
		HTTPAddress:       "0.0.0.0:4151",
		HTTPSAddress:      "0.0.0.0:4152",
		BroadcastAddress:  hostname,
		BroadcastTCPPort:  0,
		BroadcastHTTPPort: 0,

		NSQLookupdTCPAddresses: make([]string, 0),
		AuthHTTPAddresses:      make([]string, 0),

		HTTPClientConnectTimeout: 2 * time.Second,
		HTTPClientRequestTimeout: 5 * time.Second,

		MemQueueSize:    10000,
		MaxBytesPerFile: 100 * 1024 * 1024,
		SyncEvery:       2500,
		SyncTimeout:     2 * time.Second,

		QueueScanInterval:        100 * time.Millisecond,
		QueueScanRefreshInterval: 5 * time.Second,
		QueueScanSelectionCount:  20,
		QueueScanWorkerPoolMax:   4,
		QueueScanDirtyPercent:    0.25,

		MsgTimeout:    60 * time.Second,
		MaxMsgTimeout: 15 * time.Minute,
		MaxMsgSize:    1024 * 1024,
		MaxBodySize:   5 * 1024 * 1024,
		MaxReqTimeout: 1 * time.Hour,
		ClientTimeout: 60 * time.Second,

		MaxHeartbeatInterval:   60 * time.Second,
		MaxRdyCount:            2500,
		MaxOutputBufferSize:    64 * 1024,
		MaxOutputBufferTimeout: 30 * time.Second,
		MinOutputBufferTimeout: 25 * time.Millisecond,
		OutputBufferTimeout:    250 * time.Millisecond,
		MaxChannelConsumers:    0,

		StatsdPrefix:        "nsq.%s",
		StatsdInterval:      60 * time.Second,
		StatsdMemStats:      true,
		StatsdUDPPacketSize: 508,

		E2EProcessingLatencyWindowTime: time.Duration(10 * time.Minute),

		DeflateEnabled:  true,
		MaxDeflateLevel: 6,
		SnappyEnabled:   true,

		TLSMinVersion: tls.VersionTLS10,
	}
}
