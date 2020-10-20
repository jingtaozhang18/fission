# 基于Minikube的Fission 及其依赖的基础环境部署

## 部署使用的命令版本如下

| command  |  version   |
| :------: | :--------: |
| minikube |  v1.14.0   |
| kubectl  |  v1.18.5   |
|   helm   |   v3.2.4   |
|  ktctl   | 0.0.13-rc7 |


## 部署流程

### 创建minikube集群

创建集群之前，需要先确认创建集群的**名称**`CONTEXT_NAME`和集群使用的虚拟化技术`--driver`，具体细节请参考minikube的[官网](https://kubernetes.io/zh/docs/setup/learning-environment/minikube/) 。

```shell
deploy$ make start_minikube
```

### 获取集群信息

创建好集群之后，使用如下命令获取集群的`DOCKER_TLS_VERIFY`、`DOCKER_HOST`、`DOCKER_CERT_PATH`和`MINIKUBE_ACTIVE_DOCKERD`信息，对应修改Makefile中的变量

```shell
deploy$ minikube docker-env
```

### 创建基础环境和Fission 环境

现在可以通过以下命令完成基础设施环境和Fission环境的复现，具体完成的任务如下：

```shell
deploy$ make init_fission_cluster
```

部署minikube的基础设施环境，包括：

* 本地存储类
* NFS存储类
* 注册到Rancher中
* Kafka
* Keda

*ps：如果有PVC创建失败的问题，可以手动设置一下默认存储类为`local-path`*

编译Fission的组件镜像，包括：

* Bundle

* Fetcher

* Preupgradechecks

* Builder

* Cli

* FluentdWrapper

* DataFlowVisualizationFrontend

* DataFlowVisualizationBackend

编译Fission的ENV环境，包括：

* Python

使用helm3部署Fission环境，并创建一个Python的ENV环境。

*ps：部署脚本只兼容`fission-all`，对`fission-core`暂时不兼容*

## 开发工具

在对某个组件的代码进行修改的时候，可以采用本地运行该组件，使用`ktctl`工具替换掉集群中对应的组件，将流量导向本地运行的组件，同时使用http代理将本地运行的组件的流量导回集群中。参考命令：

```bash
deploy$ make kt_vpn
deploy$ make kt_exchange_executor
```

同时提供推送全部改动过的fission镜像到仓库中，并通过仓库中的镜像在其他的集群中部署Fission服务。相关命令如下：

``` bash
deploy$ make push_fission_images
deploy$ make install_fission_public
```

*脚本中的仓库地址`ml.jingtao.fun`是jingtao的私有仓库，但是暂时没有认证保护，请大佬们手下留情，之后会设置密码限制推送*

