package hooks

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	serviceLabel  = "gitwatcher.rio.cattle.io/service"
	logTokenLabel = "gitwatcher.rio.cattle.io/log-token"

	containerName = "step-build-and-push"
)

type logsHandler struct {
	core corev1.CoreV1Interface
}

func (h logsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")

	query := req.URL.Query()
	token := query.Get("log-token")

	if token == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("token is required"))
		return
	}

	if len(parts) != 3 {
		rw.WriteHeader(http.StatusUnprocessableEntity)
		rw.Write([]byte("invalid request path"))
		return
	}

	ns, name := parts[1], parts[2]
	pods, err := h.core.Pods(ns).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s, %s=%s", serviceLabel, name, logTokenLabel, token),
	})
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if len(pods.Items) == 0 {
		return
	}

	pod := pods.Items[0]
	logReq := h.core.Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Follow:    true,
		Container: containerName,
	})

	reader, err := logReq.Stream()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	defer reader.Close()

	rw.Header().Set("Content-Type", "text/plain")
	if _, err := io.Copy(rw, reader); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
}
