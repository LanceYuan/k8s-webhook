module "k8s-webhook"

require (
	github.com/gin-gonic/gin v1.7.7
	k8s.io/api v0.19.16
	k8s.io/apimachinery v0.19.16
)