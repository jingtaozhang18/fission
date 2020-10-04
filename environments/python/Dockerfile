FROM alpine

WORKDIR /app

ENV TZ Asia/Shanghai

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add --update gcc && \
    apk upgrade --no-cache && \
    apk add --no-cache python3 python3-dev build-base libev-dev libffi-dev bash py-pip git && \
    pip3 install --upgrade pip && \
    rm -r /root/.cache

COPY pip.conf /root/.pip/pip.conf
COPY requirements.txt /app
RUN pip3 install -r requirements.txt

RUN cd /tmp && git clone https://git.jingtao.fun/jingtao/prometheus_client_python.git && cd prometheus_client_python && python3 setup.py install

COPY . /app

ENTRYPOINT ["python3"]
CMD ["server.py"]
