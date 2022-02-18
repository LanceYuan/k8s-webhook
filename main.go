package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"log"
	"net/http"
	"strings"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func validation(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	req := ar.Request
	resp := &v1.AdmissionResponse{
		Allowed: false,
		Result:  &metav1.Status{},
	}
	switch req.Kind.Kind {
	case "Deployment":
		var dep appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &dep); err != nil {
			resp.Result.Message = err.Error()
			resp.Result.Code = http.StatusInternalServerError
			return resp
		}
		if strings.HasPrefix(dep.ObjectMeta.Name, "byt") || strings.HasSuffix(dep.ObjectMeta.Name, "bayantu") {
			resp.Allowed = true
			return resp
		}
		resp.Result.Code = http.StatusNotAcceptable
		return resp
	default:
		resp.Allowed = true
		return resp
	}
}

func mutation(body []byte) ([]byte, error) {
	admReview := v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
	}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}
	ar := admReview.Request
	resp := &v1.AdmissionResponse{
		Allowed: false,
		Result:  &metav1.Status{},
	}
	if ar != nil {
		switch ar.Kind.Kind {
		case "Deployment":
			var dep appsv1.Deployment
			if err := json.Unmarshal(ar.Object.Raw, &dep); err != nil {
				return nil, fmt.Errorf("unable unmarshal dep json object %v", err)
			}
			resp.Allowed = true
			resp.UID = ar.UID
			pt := v1.PatchTypeJSONPatch
			resp.PatchType = &pt
			patch := fmt.Sprintf(`[{"op":"add","path":"/metadata/labels","value":{"app":"%s"}}]`, dep.ObjectMeta.Name)
			resp.Patch = []byte(patch)
			resp.Result.Status = "Success"
			admReview.Response = resp
			return json.Marshal(admReview)
		}
	}
	return nil, errors.New("request failure")
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func main() {
	gin.SetMode(gin.ReleaseMode)
	app := gin.Default()
	app.GET("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})
	app.POST("/validate", func(ctx *gin.Context) {
		var admissionResponse *v1.AdmissionResponse
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			log.Println(err)
			return
		}
		contentType := ctx.Request.Header.Get("Content-Type")
		if contentType != "application/json" {
			ctx.String(http.StatusUnsupportedMediaType, "invalid Content-Type, expect application/json")
			return
		}
		ar := v1.AdmissionReview{}
		if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
			admissionResponse = &v1.AdmissionResponse{
				Result: &metav1.Status{
					Code:    http.StatusNotAcceptable,
					Message: err.Error(),
				},
			}
		} else {
			admissionResponse = validation(&ar)
		}
		admissionReview := v1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AdmissionReview",
				APIVersion: "admission.k8s.io/v1",
			},
		}
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
	app.POST("/mutate", func(ctx *gin.Context) {
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			log.Println(err)
			return
		}
		resp, err := mutation(body)
		if err != nil {
			log.Println(err)
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
