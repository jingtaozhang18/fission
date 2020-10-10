#!/usr/bin/env python
import json
import os
import time

import requests
from flask import Flask, request
from gevent.pywsgi import WSGIServer

PROMETHEUS_SERVER = "http://{}/api/v1/query".format(os.environ.get("PROMETHEUS_SERVER_URL", "fission-prometheus-server.fission"))


class FuncApp(Flask):
    def __init__(self, name):
        super(FuncApp, self).__init__(name)

        @self.route('/fission-build-in-funcs/fission-data-flow', methods=['POST'])
        def load():
            params = request.json
            default_params = {
                "query": "sum(rate(fission_flow_recorder_by_router[{}])) by (source, destination, stype, dtype, method, code) * 60".format(params.get("step", "5m")),
                "time": int(time.time())
            }
            default_params.update(params)
            self.logger.info("router: " + json.dumps(default_params))
            resp_router = requests.get(PROMETHEUS_SERVER, params=default_params)
            resp_router = json.loads(resp_router.text)
            default_params = {
                "query": "sum(rate(fission_flow_recorder_by_env_total[{}])) by (source, destination, stype, dtype, method, code) * 60".format(
                    params.get("step", "5m")),
                "time": int(time.time())
            }
            default_params.update(params)
            self.logger.info("env: " + json.dumps(default_params))
            resp_env = requests.get(PROMETHEUS_SERVER, params=default_params)
            resp_env = json.loads(resp_env.text)
            result = {
                "status_router": "success",
                "status_env": "success",
                "result": []
            }
            if resp_router.get("status", "fail") == "success":
                result['result'] = result['result'] + (resp_router.get("data", {}).get("result", []))
            else:
                result['status_router'] = "fail"
            if resp_env.get("status", "fail") == "success":
                result['result'] = result['result'] + (resp_env.get("data", {}).get("result", []))
            else:
                result['status_router'] = "fail"
            return json.dumps(result)


app = FuncApp(__name__)

app.logger.info("Starting gevent based server")
svc = WSGIServer(('0.0.0.0', 8888), app)
svc.serve_forever()
