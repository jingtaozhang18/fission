# 基于Minikube的Fission 及其依赖的基础环境部署

## 部署使用的命令版本如下

| command  | version |
| :------: | :-----: |
| minikube | v1.12.3 |
| kubectl  | v1.18.5 |
|   helm   | v3.2.4  |


## 部署流程

### 创建minikube集群

创建集群之前，需要先确认创建集群的**名称**`CONTEXT_NAME`和集群使用的虚拟化技术`--driver`，具体细节请参考minikube的[官网](https://kubernetes.io/zh/docs/setup/learning-environment/minikube/) 。

```shell
deploy$ make start_minikube
```

*ps：此版本的minikube不能有效的设置集群的内存大小，可以通过停止集群，在虚拟机管理软件上手动调整内存的方式，来更改实际内存大小。*

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

编译Fission的ENV环境，包括：

* Python

使用helm3部署Fission环境，并创建一个Python的ENV环境。

*ps：部署脚本只兼容`fission-all`，对`fission-core`暂时不兼容*