# k8s-webhook

## build image
```shell script
docker buildx build -t lanceyuan/k8s-webhook:v0.0.1 .
docker push lanceyuan/k8s-webhook:v0.0.1
```

## deploy
```shell script
cfssl gencert -ca=ca.crt -ca-key=ca.key -config=config.json -profile=kubernetes web-hook.json | cfssljson -bare k8s-webhook
kubectl create secret tls k8s-cert --cert=/etc/kubernetes/ssl/k8s-webhook.pem --key=/etc/kubernetes/ssl/k8s-webhook-key.pem -n kube-system
kubectl apply -f k8s-webhook.yaml
kubectl apply -f demo-validate.yaml
kubectl label namespace pro k8s-webhook=enabled
```