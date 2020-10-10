FISSION_NAMESPACE="fission"
KEDA_NAMESPACE="keda"
KAFKA_NAMESPACE="development"
# 创建的minikube 集群名称，可根据实际情况修改
CONTEXT_NAME="minikube-fission"
TEMPLATE_NAME="fission"
# 使用到的全局代理，请根据实际情况修改
PROXY=http://hp.jingtao.fun:1080
TODAY=$(shell date "+%Y_%m_%d")
PROJECT_DIR=$(shell pwd)/..

# docker environment from `minikube docker-env`，请根据实际情况修改
DOCKER_TLS_VERIFY="1"
DOCKER_HOST="tcp://192.168.39.129:2376"
DOCKER_CERT_PATH="/home/jingtao/.minikube/certs"
MINIKUBE_ACTIVE_DOCKERD="minikube-fission"

DEFAULT_TAG="$(TODAY)-1"
FISSION_IMAGE_NAME="jingtao/fission-bundle"
FISSION_IMAGE_TAG=$(DEFAULT_TAG)  # preUpgradeChecksImage 也使用这个tag
FISSION_BUILDER_IMAGE_NAME="jingtao/fission-builder"
FISSION_BUILDER_IMAGE_TAG=$(DEFAULT_TAG)
FISSION_FETCHER_IMAGE_NAME="jingtao/fission-fetcher"
FISSION_FETCHER_IMAGE_TAG=$(DEFAULT_TAG)
FISSION_PREUPGRADECHECKS_IMAGE_NAME="jingtao/fission-preupgradechecks"
FISSION_PREUPGRADECHECKS_IMAGE_TAG=$(DEFAULT_TAG)
FISSION_FLUENTD_WRAPPER_IMAGE_NAME="jingtao/fission-fluentd-wrapper"
FISSION_FLUENTD_WRAPPER_IMAGE_TAG=$(DEFAULT_TAG)

FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME="jingtao/fission_data_flow_visualization_frontend"
FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG=$(DEFAULT_TAG)
FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME="jingtao/fission_data_flow_visualization_backend"
FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG=$(DEFAULT_TAG)

FISSION_ENV_PYTHON_IMAGE_NAME="env/python"
FISSION_ENV_PYTHON_IMAGE_TAG=$(DEFAULT_TAG)


## minikube的运维命令
minikube_start:
	HTTPS_PROXY=$(PROXY) HTTP_PROXY=$(PROXY) NO_PROXY=localhost,127.0.0.1,10.96.0.0/12,192.168.99.0/24,192.168.39.0/24,192.168.50.0/24 \
		minikube start \
			--profile $(CONTEXT_NAME) \
			--docker-env http_proxy=$(PROXY) \
			--docker-env https_proxy=$(PROXY) \
			--docker-env no_proxy=localhost,127.0.0.1,10.96.0.0/12,192.168.99.0/24,192.168.39.0/24,192.168.50.0/24 \
			--addons metrics-server \
			--kubernetes-version v1.18.3 \
			--driver kvm2 \
			--cpus 8 \
			--memory 8g \
			--disk-size 40g \
			--container-runtime docker

minikube_dashboard:
	minikube dashboard --profile $(CONTEXT_NAME)

minikube_stop:
	minikube stop --profile $(CONTEXT_NAME)

minikube_delete:
	minikube delete --profile $(CONTEXT_NAME)


## fission 集群初始化
init_fission_cluster: minikube_environment_init install_fission fission_env_init


## minikube的基础设施环境
minikube_environment_init: install_localVolume_provisioner install_nfs_provisioner rancher_import install_kafka install_keda

# 本地存储类，方便单节点时数据卷的自动创建，在yaml文件中包含数据卷在节点上的创建位置，可根据实际情况修改
install_localVolume_provisioner:
	kubectl --context $(CONTEXT_NAME) apply -f infrastructures/rancher-local-path-storage.yaml

# nfs存储类，方便挂载本机的目录，nfs的服务端需要根据自己的需求搭建
install_nfs_provisioner:
	# https://github.com/helm/charts/tree/master/stable/nfs-client-provisioner
	kubectl --context $(CONTEXT_NAME) create namespace storage-nfs
	https_proxy=$(PROXY) \
		helm repo add stable https://kubernetes-charts.storage.googleapis.com && \
		helm repo update && \
		helm --kube-context $(CONTEXT_NAME) install --namespace storage-nfs --name-template helm-nfs-provisioner --set nfs.server=hp.jingtao.fun --set nfs.path=/k8s \
		--set nfs.mountOptions="{vers=4,port=2049}" \
		stable/nfs-client-provisioner

# 注册集群到rancher中，请根据实际情况修改
rancher_import:
	kubectl apply -f https://rancher.huxiang.pro/v3/import/p4zn9djk94bhl94sfnwh2xmt269lq58flpl42ltvbl75pxjgsn94fl.yaml

# 安装消息队列组件Kafka
install_kafka:
	kubectl --context $(CONTEXT_NAME) create namespace $(KAFKA_NAMESPACE)
	https_proxy=$(PROXY) helm repo add incubator http://storage.googleapis.com/kubernetes-charts-incubator
	https_proxy=$(PROXY) helm --kube-context $(CONTEXT_NAME) install --namespace $(KAFKA_NAMESPACE) -f infrastructures/kafka-values.yaml \
		--name-template my-kafka incubator/kafka

# 安装基于事件驱动的自动缩放组件Keda
install_keda:
	https_proxy=$(PROXY) helm repo add kedacore https://kedacore.github.io/charts && https_proxy=$(PROXY) helm repo update
	kubectl --context $(CONTEXT_NAME) create namespace $(KEDA_NAMESPACE)
	helm --kube-context $(CONTEXT_NAME) install keda kedacore/keda --namespace $(KEDA_NAMESPACE)

## fission的安装
install_fission: build_all
	kubectl --context $(CONTEXT_NAME) create namespace $(FISSION_NAMESPACE)
	cd $(PROJECT_DIR)/charts/fission-all && https_proxy=http://hp.jingtao.fun:1080 helm dependency update && cd -
	https_proxy=$(PROXY) \
		helm --kube-context $(CONTEXT_NAME) install --namespace $(FISSION_NAMESPACE) --name-template $(TEMPLATE_NAME) \
		 	-f minikube-fission-values.yaml \
		 	--set pullPolicy=Never \
		 	--set repository= \
		 	--set logger.fluentdImageRepository= \
		 	--set image=$(FISSION_IMAGE_NAME),imageTag=$(FISSION_IMAGE_TAG) \
		 	--set fetcher.image=$(FISSION_FETCHER_IMAGE_NAME),fetcher.imageTag=$(FISSION_FETCHER_IMAGE_TAG) \
		 	--set logger.fluentdImage=$(FISSION_FLUENTD_WRAPPER_IMAGE_NAME),logger.fluentdImageTag=$(FISSION_FLUENTD_WRAPPER_IMAGE_TAG) \
		 	--set preUpgradeChecks.image=$(FISSION_PREUPGRADECHECKS_IMAGE_NAME),preUpgradeChecks.tag=$(FISSION_PREUPGRADECHECKS_IMAGE_TAG) \
		 	--set dataFlowVisualization.frontend.image=$(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME),dataFlowVisualization.frontend.tag=$(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) \
		 	--set dataFlowVisualization.backend.image=$(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME),dataFlowVisualization.backend.tag=$(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) \
			$(PROJECT_DIR)/charts/fission-all
	echo "grafana password:"
	kubectl --context $(CONTEXT_NAME) get secret --namespace $(FISSION_NAMESPACE) fission-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo

update_fission_image: build_all
	cd $(PROJECT_DIR)/charts/fission-all && https_proxy=http://hp.jingtao.fun:1080 helm dependency update && cd -
	https_proxy=$(PROXY) \
		helm --kube-context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) upgrade $(TEMPLATE_NAME) \
			-f minikube-fission-values.yaml \
			--set pullPolicy=Never \
			--set repository= \
			--set logger.fluentdImageRepository= \
			--set image=$(FISSION_IMAGE_NAME),imageTag=$(FISSION_IMAGE_TAG) \
			--set fetcher.image=$(FISSION_FETCHER_IMAGE_NAME),fetcher.imageTag=$(FISSION_FETCHER_IMAGE_TAG) \
			--set logger.fluentdImage=$(FISSION_FLUENTD_WRAPPER_IMAGE_NAME),logger.fluentdImageTag=$(FISSION_FLUENTD_WRAPPER_IMAGE_TAG) \
			--set preUpgradeChecks.image=$(FISSION_PREUPGRADECHECKS_IMAGE_NAME),preUpgradeChecks.tag=$(FISSION_PREUPGRADECHECKS_IMAGE_TAG) \
			--set dataFlowVisualization.frontend.image=$(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME),dataFlowVisualization.frontend.tag=$(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) \
			--set dataFlowVisualization.backend.image=$(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME),dataFlowVisualization.backend.tag=$(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) \
			$(PROJECT_DIR)/charts/fission-all


## fission镜像管理
build_all: build_fission_bundle build_fission_fetcher build_fission_preupgradechecks build_fission_builder build_fluentd_wrapper build_data_flow_visualization_frontend build_data_flow_visualization_backend build_fission_cli


build_fission_bundle:
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_IMAGE_NAME):$(FISSION_IMAGE_TAG) -f $(PROJECT_DIR)/cmd/fission-bundle/Dockerfile.fission-bundle $(PROJECT_DIR)

build_fission_fetcher:
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_FETCHER_IMAGE_NAME):$(FISSION_FETCHER_IMAGE_TAG) -f $(PROJECT_DIR)/cmd/fetcher/Dockerfile.fission-fetcher $(PROJECT_DIR)

build_fission_preupgradechecks:
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_PREUPGRADECHECKS_IMAGE_NAME):$(FISSION_PREUPGRADECHECKS_IMAGE_TAG) -f $(PROJECT_DIR)/cmd/preupgradechecks/Dockerfile.fission-preupgradechecks $(PROJECT_DIR)

build_fission_builder:
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_BUILDER_IMAGE_NAME):$(FISSION_BUILDER_IMAGE_TAG) -f $(PROJECT_DIR)/cmd/builder/Dockerfile.fission-builder $(PROJECT_DIR)

build_fluentd_wrapper:
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_FLUENTD_WRAPPER_IMAGE_NAME):$(FISSION_FLUENTD_WRAPPER_IMAGE_TAG) -f $(PROJECT_DIR)/cmd/fluentd-wrapper/Dockerfile.fluentd-wrapper $(PROJECT_DIR)

build_data_flow_visualization_frontend:
	cd $(PROJECT_DIR)/cmd/data-flow-visualization/app-frontend && \
		npm run build && \
		DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME):$(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) -f Dockerfile.data-flow-visualization-frontend .

build_data_flow_visualization_backend:
	cd $(PROJECT_DIR)/cmd/data-flow-visualization/app-backend && \
		DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME):$(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) -f Dockerfile.data-flow-visualization-backend .

# 编译fission CLI到某个位置，可根据实际情况修改
build_fission_cli:
	cd $(PROJECT_DIR)/cmd/fission-cli && go build -o ~/go/bin/fission


## fission 环境部署
fission_env_init:
	cd $(PROJECT_DIR)/environments/python && \
		DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_ENV_PYTHON_IMAGE_NAME):$(FISSION_ENV_PYTHON_IMAGE_TAG) . && \
		fission --kube-context $(CONTEXT_NAME) env create --name python --image $(FISSION_ENV_PYTHON_IMAGE_NAME):$(FISSION_ENV_PYTHON_IMAGE_TAG) --builder fission/python-builder:latest --graceperiod 10

## fission 环境更新
fission_env_update:
	cd $(PROJECT_DIR)/environments/python && \
		DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
		docker build -t $(FISSION_ENV_PYTHON_IMAGE_NAME):$(FISSION_ENV_PYTHON_IMAGE_TAG) . && \
		fission --kube-context $(CONTEXT_NAME) env update --name python --image $(FISSION_ENV_PYTHON_IMAGE_NAME):$(FISSION_ENV_PYTHON_IMAGE_TAG) --graceperiod 10


## 代理相关配置
proxy_all: proxy_fission_grafana proxy_fission_prometheus proxy_fission_pushgateway proxy_fission_router


proxy_fission_router:
	kubectl --context=$(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) port-forward service/router 9080:80 --address 0.0.0.0 &
	echo "fission router map to 0.0.0.0:9080"

proxy_fission_grafana:
	echo "grafana admin password:"
	kubectl --context $(CONTEXT_NAME) get secret --namespace $(FISSION_NAMESPACE) fission-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
	kubectl --context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) port-forward service/fission-grafana 3000 --address 0.0.0.0 &
	echo "fission grafana map to 0.0.0.0:3000"

proxy_fission_prometheus:
	kubectl --context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) port-forward service/fission-prometheus-server 9090:80 --address 0.0.0.0 &
	echo "fission prometheus map to 0.0.0.0:9090"

proxy_fission_pushgateway:
	kubectl --context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) port-forward service/fission-prometheus-pushgateway 9091 --address 0.0.0.0 &
	echo "fission pushgateway map to 0.0.0.0:9091"

proxy_fission_alertmanager:
	kubectl --context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) port-forward service/fission-prometheus-alertmanager 9093 --address 0.0.0.0 &
	echo "fission alertmanager map to 0.0.0.0:9093"


## 维护小脚本
reset_grafana_passwd:
	kubectl --context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) exec \
		$(shell kubectl --context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) get pods -n fission -l app.kubernetes.io/instance=fission,app.kubernetes.io/name=grafana -o=jsonpath="{.items[0].metadata.name}"; echo) \
		-- grafana-cli admin reset-admin-password \
		$(shell kubectl --context $(CONTEXT_NAME) --namespace $(FISSION_NAMESPACE) get secret --namespace $(FISSION_NAMESPACE) fission-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo)

# 创建到集群的socks5代理
kt_vpn:
	sudo ktctl --kubeconfig /home/jingtao/.kube/config --debug connect --method=socks5

kt_exchange_executor:
	 sudo ktctl --kubeconfig /home/jingtao/.kube/config --namespace fission exchange executor --expose 8888

kt_exchange_controller:
	sudo ktctl --kubeconfig /home/jingtao/.kube/config --namespace fission exchange controller --expose 8888:8889


## 卸载
uninstall_fission:
	helm --kube-context $(CONTEXT_NAME) uninstall $(TEMPLATE_NAME) --namespace $(FISSION_NAMESPACE)
	kubectl --kube-context $(CONTEXT_NAME) delete namespace $(FISSION_NAMESPACE)


PUBLIC_CONTEXT=minikube
PUBLIC_DEFAULT_TAG="v11"
PUBLIC_REPO="ml.jingtao.fun"
PUBLIC_FISSION_IMAGE_NAME="jingtao/fission-bundle"
PUBLIC_FISSION_IMAGE_TAG=$(PUBLIC_DEFAULT_TAG)
PUBLIC_FISSION_BUILDER_IMAGE_NAME="jingtao/fission-builder"
PUBLIC_FISSION_BUILDER_IMAGE_TAG=$(PUBLIC_DEFAULT_TAG)
PUBLIC_FISSION_FETCHER_IMAGE_NAME="jingtao/fission-fetcher"
PUBLIC_FISSION_FETCHER_IMAGE_TAG=$(PUBLIC_DEFAULT_TAG)
PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_NAME="jingtao/fission-preupgradechecks"
PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_TAG=$(PUBLIC_DEFAULT_TAG)
PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_NAME="jingtao/fission-fluentd-wrapper"
PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_TAG=$(PUBLIC_DEFAULT_TAG)

PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME="jingtao/fission_data_flow_visualization_frontend"
PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG=$(PUBLIC_DEFAULT_TAG)
PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME="jingtao/fission_data_flow_visualization_backend"
PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG=$(PUBLIC_DEFAULT_TAG)

PUBLIC_FISSION_ENV_PYTHON_IMAGE_NAME="jingtao/env/python"
PUBLIC_FISSION_ENV_PYTHON_IMAGE_TAG="v11"


## 上传镜像
push_fission_images:
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_IMAGE_NAME):$(FISSION_IMAGE_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_IMAGE_NAME):$(PUBLIC_FISSION_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_BUILDER_IMAGE_NAME):$(FISSION_BUILDER_IMAGE_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_BUILDER_IMAGE_NAME):$(PUBLIC_FISSION_BUILDER_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_FETCHER_IMAGE_NAME):$(FISSION_FETCHER_IMAGE_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_FETCHER_IMAGE_NAME):$(PUBLIC_FISSION_FETCHER_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_PREUPGRADECHECKS_IMAGE_NAME):$(FISSION_PREUPGRADECHECKS_IMAGE_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_NAME):$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_FLUENTD_WRAPPER_IMAGE_NAME):$(FISSION_FLUENTD_WRAPPER_IMAGE_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_NAME):$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME):$(FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME):$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME):$(FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME):$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker tag $(FISSION_ENV_PYTHON_IMAGE_NAME):$(FISSION_ENV_PYTHON_IMAGE_TAG) $(PUBLIC_REPO)/$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_NAME):$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_IMAGE_NAME):$(PUBLIC_FISSION_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_BUILDER_IMAGE_NAME):$(PUBLIC_FISSION_BUILDER_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_FETCHER_IMAGE_NAME):$(PUBLIC_FISSION_FETCHER_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_NAME):$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_NAME):$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME):$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME):$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) && \
	DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) DOCKER_HOST=$(DOCKER_HOST) DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) MINIKUBE_ACTIVE_DOCKERD=$(MINIKUBE_ACTIVE_DOCKERD) \
	docker push $(PUBLIC_REPO)/$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_NAME):$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_TAG)


## fission的安装
install_fission_public:
	kubectl --context $(PUBLIC_CONTEXT) create namespace $(FISSION_NAMESPACE)
	cd $(PROJECT_DIR)/charts/fission-all && https_proxy=$(PROXY) helm dependency update && cd -
	https_proxy=$(PROXY) \
		helm --kube-context $(PUBLIC_CONTEXT) install --namespace $(FISSION_NAMESPACE) --name-template $(TEMPLATE_NAME) \
		 	-f minikube-fission-values.yaml \
			--set repository=$(PUBLIC_REPO) \
			--set logger.fluentdImageRepository=$(PUBLIC_REPO) \
			--set image=$(PUBLIC_FISSION_IMAGE_NAME),imageTag=$(PUBLIC_FISSION_IMAGE_TAG) \
			--set fetcher.image=$(PUBLIC_FISSION_FETCHER_IMAGE_NAME),fetcher.imageTag=$(PUBLIC_FISSION_FETCHER_IMAGE_TAG) \
			--set logger.fluentdImage=$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_NAME),logger.fluentdImageTag=$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_TAG) \
			--set preUpgradeChecks.image=$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_NAME),preUpgradeChecks.tag=$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_TAG) \
			--set dataFlowVisualization.frontend.image=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME),dataFlowVisualization.frontend.tag=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) \
			--set dataFlowVisualization.backend.image=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME),dataFlowVisualization.backend.tag=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) \
			$(PROJECT_DIR)/charts/fission-all
	echo "grafana password:"
	kubectl --context $(CONTEXT_NAME) get secret --namespace $(FISSION_NAMESPACE) fission-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
	fission --kube-context $(PUBLIC_CONTEXT) env create --name python --image $(PUBLIC_REPO)/$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_NAME):$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_TAG) --builder fission/python-builder:latest --graceperiod 10

update_fission_image_public:
	https_proxy=$(PROXY) \
		helm --kube-context $(PUBLIC_CONTEXT) --namespace $(FISSION_NAMESPACE) upgrade $(TEMPLATE_NAME) \
			-f minikube-fission-values.yaml \
			--set repository=$(PUBLIC_REPO) \
			--set logger.fluentdImageRepository=$(PUBLIC_REPO) \
			--set image=$(PUBLIC_FISSION_IMAGE_NAME),imageTag=$(PUBLIC_FISSION_IMAGE_TAG) \
			--set fetcher.image=$(PUBLIC_FISSION_FETCHER_IMAGE_NAME),fetcher.imageTag=$(PUBLIC_FISSION_FETCHER_IMAGE_TAG) \
			--set logger.fluentdImage=$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_NAME),logger.fluentdImageTag=$(PUBLIC_FISSION_FLUENTD_WRAPPER_IMAGE_TAG) \
			--set preUpgradeChecks.image=$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_NAME),preUpgradeChecks.tag=$(PUBLIC_FISSION_PREUPGRADECHECKS_IMAGE_TAG) \
			--set dataFlowVisualization.frontend.image=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_IMAGE_NAME),dataFlowVisualization.frontend.tag=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_FRONTEND_TAG) \
			--set dataFlowVisualization.backend.image=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_IMAGE_NAME),dataFlowVisualization.backend.tag=$(PUBLIC_FISSION_DATA_FLOW_VISUALIZATION_BACKEND_TAG) \
			$(PROJECT_DIR)/charts/fission-all
	fission --kube-context $(PUBLIC_CONTEXT) env update --name python --image $(PUBLIC_REPO)/$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_NAME):$(PUBLIC_FISSION_ENV_PYTHON_IMAGE_TAG) --builder fission/python-builder:latest --graceperiod 10
