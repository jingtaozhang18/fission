# The changes by jingtao:
* 添加minikube和fission的安装、部署、升级脚本（charts文件下），可一键部署minikube基础环境和fission开发测试环境，并可进行迭代更新
* 对fission CLI的查看函数日志的功能，添加从当前时间开始读、隐藏系统时间和根据时区调整日志时间的功能
* 增加从pod中获取函数日志的方式，可以在不依赖任何日志组件的前提下，达到实时日志查看目标，方便用户自我调试
* 允许用户在配置timetrigger触发时携带body体，实现对函数更加定制化的周期性调用
* 允许所有命名空间的fission函数配置使用fission-secret-configmap命名空间下的configmap和secret，可将其作为全局默认参数来使用，减少用户在沟通集群相关参数方面的成本
* 添加可以配置的日志投递功能，并提供全局和局部的日志投递配置相结合的方式，允许用户更自由的将日志输出到各种日志收集后端
* 添加记录函数、队列之间数据流动的功能，对应添加展示数据流动的可视化组件

# Fission: Serverless Functions for Kubernetes

[![Build Status](https://travis-ci.org/fission/fission.svg?branch=master)](https://travis-ci.org/fission/fission)
[![Go Report Card](https://goreportcard.com/badge/github.com/fission/fission)](https://goreportcard.com/report/github.com/fission/fission)
[![codecov](https://codecov.io/gh/fission/fission/branch/master/graph/badge.svg)](https://codecov.io/gh/fission/fission)

[fission.io](http://fission.io) | [@fissionio](http://twitter.com/fissionio) | [Slack](https://join.slack.com/t/fissionio/shared_invite/enQtOTI3NjgyMjE5NzE3LTllODJiODBmYTBiYWUwMWQxZWRhNDhiZDMyN2EyNjAzMTFiYjE2Nzc1NzE0MTU4ZTg2MzVjMDQ1NWY3MGJhZmE)

<img src="https://docs.fission.io/images/logo.png" width="300">

Fission is a fast serverless framework for Kubernetes with a focus on
developer productivity and high performance.

Fission operates on _just the code_: Docker and Kubernetes are
abstracted away under normal operation, though you can use both to
extend Fission if you want to.

Fission is extensible to any language; the core is written in Go, and
language-specific parts are isolated in something called
_environments_ (more below).  Fission currently supports NodeJS, Python, Ruby, Go, 
PHP, Bash, and any Linux executable, with more languages coming soon.

# Performance: 100msec cold start

Fission maintains a pool of "warm" containers that each contain a
small dynamic loader.  When a function is first called,
i.e. "cold-started", a running container is chosen and the function is
loaded.  This pool is what makes Fission fast: cold-start latencies
are typically about 100msec.

# Kubernetes is the right place for Serverless

We're built on Kubernetes because we think any non-trivial app will
use a combination of serverless functions and more conventional
microservices, and Kubernetes is a great framework to bring these
together seamlessly.

Building on Kubernetes also means that anything you do for operations
on your Kubernetes cluster &mdash; such as monitoring or log
aggregation &mdash; also helps with ops on your Fission deployment.

# Getting started and documentation

## Fission Concepts

Visit [concepts](https://docs.fission.io/docs/concepts/) for more details.

## Documentations

You can learn more about Fission and get started from [Fission Docs](https://docs.fission.io/docs).
* See the [installation guide](https://docs.fission.io/docs/installation/) for installing and running Fission.
* See the [troubleshooting guide](https://docs.fission.io/docs/trouble-shooting/) for debugging your functions and Fission installation.

## Usage

```bash
  # Add the stock NodeJS env to your Fission deployment
  $ fission env create --name nodejs --image fission/node-env

  # A javascript one-liner that prints "hello world"
  $ curl https://raw.githubusercontent.com/fission/fission/master/examples/nodejs/hello.js > hello.js

  # Upload your function code to fission
  $ fission function create --name hello --env nodejs --code hello.js

  # Map GET /hello to your new function
  $ fission route create --method GET --url /hello --function hello

  # Run the function.  This takes about 100msec the first time.
  $ fission function test --name hello
  Hello, world!
```

# Contributing

## Building Fission
See the [compilation guide](https://docs.fission.io/docs/contributing/).

## Contact
Fission is a project by [many contributors](https://github.com/fission/fission/graphs/contributors).
Reach us on [slack](https://join.slack.com/t/fissionio/shared_invite/enQtOTI3NjgyMjE5NzE3LTllODJiODBmYTBiYWUwMWQxZWRhNDhiZDMyN2EyNjAzMTFiYjE2Nzc1NzE0MTU4ZTg2MzVjMDQ1NWY3MGJhZmE) or [twitter](https://twitter.com/fissionio).

## Community Meeting 

A regular community meeting takes place every other Thursday at 09:00 AM PT (Pacific Time). [Convert to your local timezone](http://www.thetimezoneconverter.com/?t=09:00&tz=PT%20%28Pacific%20Time%29).

Meeting Link: https://zoom.us/j/413921817 

The meeting agenda for next meeting and notes from past meetnigs are maintained in [this document](https://docs.google.com/document/d/1E-xw4KJgka4sUpETHxr9BJBYntzrtxlAN_CE3Wt8kws). You are welcome to join to discuss direction of project, design and implementation reviews and general questions about project etc.

# Official Releases

Official releases of Fission can be found on [the releases page](https://github.com/fission/fission/releases). 
Please note that it is strongly recommended that you use official releases of Fission, as unreleased versions from 
the master branch are subject to changes and incompatibilities that will not be supported in the official releases. 
Builds from the master branch can have functionality changed and even removed at any time without compatibility support 
and without prior notice.

# Sponsors
The following companies, organizations, and individuals support Fission's ongoing maintenance and development.
Become a sponsor to get your logo on our README on Github with a link to your site.

<p>
    <a href="https://infracloud.io/"><img src="https://fission.io/sponsors/infracloud.png" alt="InfraCloud" height="70"></a>
    <a href="https://srcmesh.com/"><img src="https://fission.io/sponsors/srcmesh.png" alt="Srcmesh" height="70"></a>
</p>

# Licensing

Fission is under the Apache 2.0 license.
