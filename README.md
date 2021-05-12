## Nsq annotation

### 背景
NSQ是golang实现的消息分发平台，支持分布式和高可用。
其良好的性能可以每天分发超过十亿多的消息。其他的类似消息队列有RabbitMq，
Kafka等，他们与NSQ各有功能上的不同和实现上的不同。NSQ可以作为有golang背景的同学，来学习的第一个消息队列。
希望这个项目可以帮到你，请帮忙Star一下，谢谢。

### 版本
代码使用的是2021.05.12的Master分支。

### 目录结构
>[√] 代表已经完成备注。
>
>[×] 代表还未全部完成备注。
>
>[-] 代表无需备注，自行阅读。

````
├── apps
│   ├── nsq_stat
│   ├── nsq_tail
│   ├── nsq_to_file
│   ├── nsq_to_http
│   ├── nsq_to_nsq
│   ├── nsqadmin
│   ├── nsqd                     [√] nsqd程序入口
│   ├── nsqlookupd
│   └── to_nsq
├── bench
│   ├── bench_channels
│   ├── bench_reader
│   └── bench_writer
├── contrib
├── internal
│   ├── app
│   ├── auth
│   ├── clusterinfo
│   ├── dirlock
│   ├── http_api
│   ├── lg
│   ├── pqueue
│   ├── protocol
│   ├── quantile
│   ├── statsd
│   ├── stringy
│   ├── test
│   ├── util
│   ├── version
│   └── writers
├── nsqadmin
│   ├── static
│   │   ├── css
│   │   ├── fonts
│   │   ├── html
│   │   ├── img
│   │   └── js
│   │       ├── collections
│   │       ├── lib
│   │       ├── models
│   │       └── views
│   └── test
├── nsqd
│   └── test
│       └── certs
└── nsqlookupd
