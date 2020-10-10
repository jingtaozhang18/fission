const {createProxyMiddleware} = require('http-proxy-middleware')
// 在npm run build 之后失效，需要依靠nginx的反代理转发。 此配置仅用于测试时使用
module.exports = function (app) {
    app.use(
        createProxyMiddleware('/apis/*', {
            target: 'http://192.168.39.129:31314',
            changeOrigin: true,
        })
    )
}