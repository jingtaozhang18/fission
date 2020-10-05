#!/usr/bin/env python
import importlib
import logging
import os
import sys
import threading
import time

import redis
import requests
from flask import Flask, request, abort, g
from gevent.pywsgi import WSGIServer
from kafka import KafkaProducer
from prometheus_client import PrometheusForFission
import bjoern
import sentry_sdk
from sentry_sdk.integrations.flask import FlaskIntegration


IS_PY2 = (sys.version_info.major == 2)
SENTRY_DSN = os.environ.get('SENTRY_DSN', None)
SENTRY_RELEASE = os.environ.get('SENTRY_RELEASE', None)

if SENTRY_DSN:
    params = {
        'dsn': SENTRY_DSN,
        'integrations': [FlaskIntegration()]
    }
    if SENTRY_RELEASE:
        params['release'] = SENTRY_RELEASE
    sentry_sdk.init(**params)

PATH_CONFIGS = "/configs"
PATH_SECRETS = "/secrets"
PUSHGATEWAY_URL_DEFAULT = "fission-prometheus-pushgateway.fission:9091"  # may be overwritten by configs

GLOBAL_CONFIG_KEY = "global"  # 全局配置文件别名
LOCAL_CONFIG_KEY = "local"  # 局部配置文件别名

FISSION_ROUTER_TEMPLATE_KEY = "fission-router-template"  # 可以通过全局配置文件中的FISSION_ROUTER_TEMPLATE_KEY自定义fission的路由位置模板
FISSION_ROUTER_KEY = "fission-router"  # 可以通过全局配置文件中的FISSION_ROUTER_KEY自定义fission的路由位置
FISSION_FLOW_KEY = "fission-flow"  # 可以通过全局配置文件中的FISSION_FLOW_KEY自定义fission流量使用的key

FISSION_FLOW_DEFAULT_VALUE = "fission_flow_recorder_by_env"  # 默认记录fission流量使用的key

FISSION_TYPE_FUNC = "func"
FISSION_TYPE_KAFKA = "kafka"


def import_src(path):
    if IS_PY2:
        import imp
        return imp.load_source('mod', path)
    else:
        # the imp module is deprecated in Python3. use importlib instead.
        return importlib.machinery.SourceFileLoader('mod', path).load_module()


def synchronized(func):
    """
    a simple function lock
    """
    func.__lock__ = threading.Lock()

    def lock_func(*args, **kwargs):
        with func.__lock__:
            return func(*args, **kwargs)

    return lock_func


class Info(object):
    """
    for Cache class
    """

    def __init__(self, value, timeout):
        """
        存储的内容和过期的时间
        :param value:
        :param timeout:
        """
        self.value = value
        self.timeout = timeout


class Cache(object):
    """cache"""

    def __init__(self):
        self.content = dict()  # key: Info

    @synchronized
    def put(self, key, func=lambda x: x, param=None, timeout=0, use_old=False, old_timeout=30):
        """
        存放信息
        :param key: 键
        :param func: 获取值的函数
        :param param: func的参数
        :param timeout: 过期时间，单位秒，0表示永不过期
        :param use_old: 当数据超时且获取新值的函数没有成功，是否返回超时了的旧数据
        :param old_timeout: 超时的数据可以继续存活的时间
        :return:
        """
        now = time.time()
        timeout += now if timeout != 0 else 0
        old_timeout += now
        value = self.get(key, no_delete=use_old)
        if value is not None:
            return value
        value = func(param)
        if value is None and use_old:
            if key in self.content:
                info = self.content[key]
                if info.timeout != 0:
                    info.timeout = old_timeout
                return info.value
        else:
            self.content[key] = Info(value, timeout)
        return value

    @synchronized
    def pop(self, key):
        if key in self.content:
            self.content.pop(key)

    def get(self, key, no_delete=False):
        """
        读取信息
        :param key: 键
        :param no_delete: 不执行删除操作
        """
        if key not in self.content:
            return None
        info = self.content[key]
        if info.timeout > time.time() or info.timeout == 0:
            return info.value
        else:
            if no_delete is False:
                self.pop(key)
            return None

    def get_and_write(self, key: str, func=lambda x: x, param=None, timeout=0, use_old=True, old_timeout=30):
        ans = self.get(key, no_delete=True)
        if ans is not None:
            return ans
        return self.put(key, func, param, timeout, use_old, old_timeout)


def add_params(con, path, key, value):
    """
    在字典中添加内容，path 是字典的路径，key是key
    """
    pos = con
    for p in path:
        if p not in pos:
            pos[p] = {}
        pos = pos[p]
    pos[key] = value


def read_config(base_dir, fns, fn):
    """读取目录下的配置文件"""
    walks = os.walk(base_dir)
    configs = dict()
    for current_path, dir_list, file_list in walks:  # BFS
        for file_name in file_list:
            paths = current_path.split("/")[2:]  # 第一个是空，第二个是config或者secrets
            value = open(os.path.join(current_path, file_name)).read()  # 读取文件中的内容
            add_params(configs, paths, file_name, value)
    # set alias, make it easy for user to get the parameters
    if "fission-secret-configmap" in configs:
        configs[GLOBAL_CONFIG_KEY] = configs["fission-secret-configmap"].get("fission-function-global-configmap", {})
    local_key = "func-{}".format(fn)
    if fns in configs and local_key in configs.get(fns, {}):
        configs[LOCAL_CONFIG_KEY] = configs.get(fns, {}).get(local_key, {})
    return configs


class FuncApp(Flask):
    def __init__(self, name, loglevel=logging.DEBUG):
        super(FuncApp, self).__init__(name)

        # init the class members
        self.func_namespace = None  # 用户函数所在的命名空间
        self.func_name = None  # 用户函数名称
        self.func_updateTime = None  # 函数最近更新的时间
        self.userfunc = None  # 用户函数句柄
        self.metric_handler = None  # 埋点上报句柄
        self.kafkaProducer_handler = None  # kafka 生产客户端
        self.redis_handler = None  # redis 客户端句柄
        self.configs = {}  # 函数configmap信息
        self.secrets = {}  # 函数secrets信息
        self.cache = None  # Cache() # pod 周期级缓存对象
        self.logger.setLevel(loglevel)  # 设置日志的默认级别，在加载函数时会根据用户的环境变量进行修改

        @self.route('/specialize', methods=['POST'])
        def load():
            self.logger.info('env_info: /specialize called')
            # load user function from codepath
            self.userfunc = import_src('/userfunc/user').main
            return ""

        @self.route('/v2/specialize', methods=['POST'])
        def loadv2():
            body = request.get_json()
            filepath = body['filepath']
            handler = body['functionName']
            self.logger.info('/v2/specialize called with  filepath = "{}"   handler = "{}"'.format(filepath, handler))

            # handler looks like `path.to.module.function`
            parts = handler.rsplit(".", 1)
            if len(handler) == 0:
                # default to main.main if entrypoint wasn't provided
                moduleName = 'main'
                funcName = 'main'
            elif len(parts) == 1:
                moduleName = 'main'
                funcName = parts[0]
            else:
                moduleName = parts[0]
                funcName = parts[1]
            self.logger.debug('moduleName = "{}"    funcName = "{}"'.format(moduleName, funcName))

            # check whether the destination is a directory or a file
            if os.path.isdir(filepath):
                # add package directory path into module search path
                sys.path.append(filepath)

                self.logger.debug('__package__ = "{}"'.format(__package__))
                if __package__:
                    mod = importlib.import_module(moduleName, __package__)
                else:
                    mod = importlib.import_module(moduleName)

            else:
                # load source from destination python file
                mod = import_src(filepath)

            # load user function from module
            self.userfunc = getattr(mod, funcName)

            self.func_namespace = body.get('FunctionMetadata', {}).get('namespace', "")
            self.func_name = body.get('FunctionMetadata', {}).get('name', "")

            assert len(self.func_name) != 0 and len(self.func_namespace) != 0

            # get configs and secrets
            self.configs = read_config(PATH_CONFIGS, self.func_namespace, self.func_name)
            self.secrets = read_config(PATH_SECRETS, self.func_namespace, self.func_name)

            update_time = body.get('FunctionMetadata', {}).get('managedFields', [])
            if len(update_time) != 0:
                update_time = update_time[0].get('time', "unknown")
            else:
                update_time = body.get('FunctionMetadata', {}).get('creationTimestamp', "unknown")
            self.func_updateTime = update_time

            # 设置日志级别
            self.set_logger_level()
            # 设置prometheus客户端
            self.set_prometheus_client()
            # 设置kafka客户端
            self.set_kafka_client()
            # 设置redis客户端
            self.set_redis_client()
            # 设置缓存
            self.set_cache()

            return ""

        @self.route('/healthz', methods=['GET'])
        def healthz():
            return "", 200

        @self.route('/', methods=['GET', 'POST', 'PUT', 'HEAD', 'OPTIONS', 'DELETE'])
        def f():
            if self.userfunc is None:
                print("Generic container: no requests supported")
                abort(500)
            # use g to pass parameter to function
            g.logger = self.logger
            g.metric_handler = self.metric_handler
            g.configs = self.configs  # 函数配置的参数
            g.secrets = self.secrets
            g.cache = self.cache
            g.kafkaProducer_handler = self.kafkaProducer_handler
            g.redis_handler = self.redis_handler
            g.access_fission_func = self.access_fission_func
            g.kafka_send = self.kafka_send
            self.pre_process()
            try:
                res = self.userfunc()
            except Exception as e:
                self.post_process(excep=e)
                raise e
            self.post_process()
            return res

    def post_process(self, excep: Exception = None):
        mq_resp_topic = request.headers.get("X-Fission-MQTrigger-RespTopic", None)
        mq_error_topic = request.headers.get("X-Fission-MQTrigger-ErrorTopic", None)
        if excep is None and mq_resp_topic is not None and len(mq_resp_topic) != 0:
            labels_dict = {
                "source": ".".join([FISSION_TYPE_FUNC, self.func_namespace, self.func_name]),
                "destination": ".".join(["kafka", mq_resp_topic]),
                "stype": FISSION_TYPE_FUNC,
                "dtype": FISSION_TYPE_KAFKA,
                "method": FISSION_TYPE_KAFKA,
                "code": "unknown"  # todo 记录推送的结果
            }
            self.metric_handler.counter("", labels_dict, 1, complete_name=self.configs.get(GLOBAL_CONFIG_KEY, {}).get(FISSION_FLOW_KEY, FISSION_FLOW_DEFAULT_VALUE))
        if excep is not None and mq_error_topic is not None and len(mq_error_topic) != 0:
            labels_dict = {
                "source": ".".join([FISSION_TYPE_FUNC, self.func_namespace, self.func_name]),
                "destination": ".".join(["kafka", mq_error_topic]),
                "stype": FISSION_TYPE_FUNC,
                "dtype": FISSION_TYPE_KAFKA,
                "method": FISSION_TYPE_KAFKA,
                "code": "unknown"  # todo 记录推送的结果
            }
            self.metric_handler.counter("", labels_dict, 1, complete_name=self.configs.get(GLOBAL_CONFIG_KEY, {}).get(FISSION_FLOW_KEY, FISSION_FLOW_DEFAULT_VALUE))

    def pre_process(self):
        """
        处理请求之前先行处理
        :return:
        """
        # 将记录消息队列到函数的数据流的任务转移到router组件中完成
        pass

    def access_fission_func(self, namespace: str, name: str, method: str, url=None, params=None, data=None, json=None, **kwargs):
        # 构造fission函数的url地址
        domain = self.configs.get(GLOBAL_CONFIG_KEY, {}).get(FISSION_ROUTER_KEY, "http://router.fission")
        url = self.configs.get(GLOBAL_CONFIG_KEY, {}).get(FISSION_ROUTER_TEMPLATE_KEY, "{domain}/{namespace}/{name}") \
            .format(domain=domain, namespace=namespace, name=name) if url is None else url

        # 同步官方中的请求方式，设置kwargs选项
        if method == "get":
            kwargs.setdefault('allow_redirects', True)
        if method == "options":
            kwargs.setdefault("allow_redirects", True)
        if method == "head":
            kwargs.setdefault('allow_redirects', False)

        headers = {
            "X-Fission-Flow-Source": ".".join([FISSION_TYPE_FUNC, self.func_namespace, self.func_name]),
            "X-Fission-Flow-Source-Type": FISSION_TYPE_FUNC
        }
        if "headers" in kwargs and type(kwargs["headers"]) == dict:
            headers.update(kwargs.get("headers"))
        resp = requests.request(method, url, params=params, data=data, json=json, headers=headers, **kwargs)

        # 将记录函数到函数的数据流的任务转移到router组件中完成
        if resp.status_code != 200:
            self.logger.error("access_fission_func url: {}, status_code:{}".format(url, resp.status_code))
        else:
            self.logger.debug("access_fission_func url: {}, status_code:{}".format(url, resp.status_code))
        return resp

    def kafka_send(self, topic, value=None, key=None, headers=None, partition=None, timestamp_ms=None):
        ret = self.kafkaProducer_handler.send(topic, value, key, headers, partition, timestamp_ms)
        labels_dict = {
            "source": ".".join([FISSION_TYPE_FUNC, self.func_namespace, self.func_name]),
            "destination": ".".join(["kafka", topic]),
            "stype": FISSION_TYPE_FUNC,
            "dtype": FISSION_TYPE_KAFKA,
            "method": FISSION_TYPE_KAFKA,
            "code": "unknown"  # todo 记录推送的结果
        }
        self.metric_handler.counter("", labels_dict, 1, complete_name=self.configs.get(GLOBAL_CONFIG_KEY, {}).get(FISSION_FLOW_KEY, FISSION_FLOW_DEFAULT_VALUE))
        return ret

    def set_logger_level(self):
        """set logger level"""
        logger_level = self.configs.get(LOCAL_CONFIG_KEY, {}).get("logger_level", "debug")
        level_map = {
            "debug": logging.DEBUG,
            "info": logging.INFO,
            "warn": logging.WARN,
            "error": logging.ERROR
        }
        if logger_level not in level_map:
            self.logger.error("logger level: {} is illegal!".format(logger_level))
        self.logger.debug("logger level will be {}".format(logger_level))
        self.logger.setLevel(level_map.get(logger_level, logging.DEBUG))

    def set_prometheus_client(self):
        """set prometheus client. The default is enable it."""
        if self.configs.get(LOCAL_CONFIG_KEY, {}).get("prometheus-enabled", "y") == "n":
            self.logger.debug("the prometheus client will not be created")
            return
        pushgateway_url = self.configs.get(LOCAL_CONFIG_KEY, {}).get("pushgateway-url", "")
        if len(pushgateway_url) == 0:
            pushgateway_url = self.configs.get(GLOBAL_CONFIG_KEY, {}).get("pushgateway-url", "")
        if len(pushgateway_url) == 0:
            pushgateway_url = PUSHGATEWAY_URL_DEFAULT
        prefix = self.func_namespace + "_" + self.func_name
        self.logger.debug("pushgateway_url is {}, prefix is {}, update_time is {}".format(pushgateway_url, prefix, self.func_updateTime))
        self.metric_handler = PrometheusForFission(prefix, self.func_updateTime, pushgateway_url, self.logger)

    def set_kafka_client(self):
        """set kafka client. The default is disable it."""
        if self.configs.get(LOCAL_CONFIG_KEY, {}).get("kafka-enabled", "n") == "n":
            self.logger.debug("the kafka producer will not be created")
            return
        kafka_broker_list = self.configs.get(LOCAL_CONFIG_KEY, {}).get("kafka-broker-list", "")
        if len(kafka_broker_list) == 0:
            kafka_broker_list = self.configs.get(GLOBAL_CONFIG_KEY, {}).get("kafka-broker-list", "")
        self.logger.debug("the kafka producer will connect to {}".format(kafka_broker_list))
        self.kafkaProducer_handler = KafkaProducer(bootstrap_servers=kafka_broker_list)

    def set_redis_client(self):
        """set redis client. The default is disable it."""
        if self.configs.get(LOCAL_CONFIG_KEY, {}).get("redis-enabled", "n") == "n":
            self.logger.debug("the redis client will not be created")
            return
        redis_url = self.configs.get(LOCAL_CONFIG_KEY, {}).get("redis-url", "")
        if len(redis_url) == 0:
            redis_url = self.configs.get(GLOBAL_CONFIG_KEY, {}).get("redis-url", "")
        self.logger.debug("the redis client will connect to {}".format(redis_url))
        self.redis_handler = redis.StrictRedis.from_url(redis_url)

    def set_cache(self):
        """set cache. The default is enable it"""
        if self.configs.get(LOCAL_CONFIG_KEY, {}).get("podcache-enabled", "y") == "n":
            self.logger.debug("the cache will not be created")
            return
        self.cache = Cache()


app = FuncApp(__name__, logging.DEBUG)

#
# TODO: this starts the built-in server, which isn't the most
# efficient.  We should use something better.
#
if os.environ.get("WSGI_FRAMEWORK") == "GEVENT":
    app.logger.info("Starting gevent based server")
    svc = WSGIServer(('0.0.0.0', 8888), app)
    svc.serve_forever()
else:
    app.logger.info("Starting bjoern based server")
    bjoern.run(app, '0.0.0.0', 8888, reuse_port=True)
