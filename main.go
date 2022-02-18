package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"log"
	"net/http"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func validation(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	req := ar.Request
	fmt.Print(req)
	return &v1.AdmissionResponse{
		Allowed: true,
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	app := gin.Default()
	app.Any("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})
	app.Any("/validate", func(ctx *gin.Context) {
		var admissionResponse *v1.AdmissionResponse
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			log.Println(err)
			return
		}
		contentType := ctx.Request.Header.Get("Content-Typ")
		if contentType != "application/json" {
			ctx.String(http.StatusUnsupportedMediaType, "invalid Content-Type, expect application/json")
			return
		}
		ar := v1.AdmissionReview{}
		if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
			admissionResponse = &v1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		} else {
			admissionResponse = validation(&ar)
		}
		admissionReview := v1.AdmissionReview{}
		if admissionResponse != nil {
			admissionReview.Response = admissionResponse
			if ar.Request != nil {
				admissionReview.Response.UID = ar.Request.UID
			}
		}
		resp, err := json.Marshal(admissionReview)
		if err != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("could not write response: %v", err))
			return
		}
		if _, err := ctx.Writer.Write(resp); err != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("could not write response: %v", err))
		}
	})
	if err := app.RunTLS(":8080", "/opt/cert/tls.crt", "/opt/cert/tls.key"); err != nil {
		log.Print(err)
	}
}
