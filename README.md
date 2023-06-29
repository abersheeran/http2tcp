# http2tcp

将 HTTP 链接转换为加密 TCP 通道。参考了 [http2tcp](https://github.com/movsb/http2tcp) 的实现。

## 使用

如下命令产生的结果：服务端监听 `8080` 端口，客户端将 `8081` 端口的请求转发到服务端的 `6379` 端口。

### 服务端

```bash
./http2tcp server -l :8080 -a longlongauthtoken
```

### 客户端

```bash
./http2tcp client -s serverhost:8080 -a longlongauthtoken -l :8081 -t 127.0.0.1:6379
```

## 原理

HTTP 规范里，携带 `Upgrade` 头的请求可以将 HTTP 协议的链接转换为其他协议的链接，在服务端返回 `101` 状态码之后，链接经过的七层代理服务（例如 `nginx`）将转变为四层代理。`http2tcp` 利用这一点，将 HTTP 链接转换为加密的 TCP 通道。

## 加密

`http2tcp` 使用 `AES-GCM` 算法加密通信内容，`key` 是鉴权令牌的 `sha256` 值。
