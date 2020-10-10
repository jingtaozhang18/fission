import axios from "axios";
import G6 from "@antv/g6";

axios.defaults.baseURL = ""

async function getFlowData(time, step) {
    // const params = {time: 1600073370};
    const params = {};
    if (time !== "") {
        params["time"] = time
    }
    if (step !== "") {
        params["step"] = step
    }
    let flowData = {}

    // 同步请求
    await axios.post('/apis/fission-build-in-funcs/fission-data-flow', params, {
        headers: {
            "X-Fission-Flow-Source": "fission-front.data-flow",
            "X-Fission-Flow-Source-Type": "fission-front"
        }
    }).then(response => function (response) {
        flowData = {nodes: [], edges: []}
        // 获取所有节点的信息
        let nodes = new Set();
        let nodesInfo = {}
        for (let i in response['result']) {
            nodes.add(response['result'][i]['metric']['source'])
            nodesInfo[response['result'][i]['metric']['source']] = response['result'][i]['metric']['stype']
            nodes.add(response['result'][i]['metric']['destination'])
            nodesInfo[response['result'][i]['metric']['destination']] = response['result'][i]['metric']['dtype']
        }

        nodes.forEach(function (key, value) {
            let nodeType
            let size
            let style
            if (nodesInfo[key] === "func") { // 定义函数的形状
                nodeType = "triangle"
                size = [15, 6]
                style = {
                    lineWidth: 2,
                    stroke: '#5B8FF9',
                    fill: '#C6E5FF',
                }
            } else if (nodesInfo[key] === "kafka" || nodesInfo[key] === "kafka" || nodesInfo[key] === "azurequeue" || nodesInfo[key] === "nats") { // 定义其他消息队列的形状
                nodeType = "rect"
                size = [30, 20]
                style = {
                    lineWidth: 2,
                    stroke: '#1bde8a',
                    fill: '#12eaaa',
                }
            } else {
                // 默认的圆形
            }
            value = value.split('.')
            if (value.length > 1) {
                value.shift()
            }
            value = value.join(".")
            flowData.nodes.push(({
                id: key,
                label: value,
                type: nodeType,
                size: size,
                style: style
            }))
        })

        for (let i in response['result']) {
            if (response['result'][i]['value'][1] === "0") {
                continue
            }
            let color
            if (response['result'][i]['metric']['code'] === "unknown") {
                response['result'][i]['metric']['code'] = "0"
            }
            if (response['result'][i]['metric']['code'] === "0") {
                color = "#aab4b4"
            } else if (response['result'][i]['metric']['code'] === "200") {
                color = "#00ff00"
            } else {
                color = "#ff0000"
            }
            flowData.edges.push({
                source: response['result'][i]['metric']['source'],
                target: response['result'][i]['metric']['destination'],
                type: "line-dash",
                label: `${parseFloat(response['result'][i]['value'][1]).toFixed(2)}(code:${response['result'][i]['metric']['code']})`,
                style: {
                    size: 2,
                    stroke: color,
                    endArrow: true,
                },

            })
        }
    }(response.data))
    G6.Util.processParallelEdges(flowData.edges);
    return flowData
}

export default getFlowData