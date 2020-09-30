/*
Copyright 2016 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	_ "strings"
	"time"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/fission/fission/pkg/fission-cli/logdb"
	"github.com/go-openapi/spec"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	ferror "github.com/fission/fission/pkg/error"
)

func RegisterFunctionRoute(ws *restful.WebService) {
	tags := []string{"Function"}
	specTag = append(specTag, spec.Tag{TagProps: spec.TagProps{Name: "Function", Description: "Function Operation"}})

	ws.Route(
		ws.GET("/v2/functions").
			Doc("List all functions").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.QueryParameter("namespace", "Namespace of function").DataType("string").DefaultValue(metav1.NamespaceAll).Required(false)).
			Produces(restful.MIME_JSON).
			Writes([]fv1.Function{}).
			Returns(http.StatusOK, "List of functions", []fv1.Function{}))

	ws.Route(
		ws.POST("/v2/functions").
			Doc("Create function").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Produces(restful.MIME_JSON).
			Reads(fv1.Function{}).
			Writes(metav1.ObjectMeta{}).
			Returns(http.StatusOK, "ObjectMeta of created function", metav1.ObjectMeta{}))

	ws.Route(
		ws.GET("/v2/functions/{function}").
			Doc("Get detail of function").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.PathParameter("function", "Function name").DataType("string").DefaultValue("").Required(true)).
			Param(ws.QueryParameter("namespace", "Namespace of function").DataType("string").DefaultValue(metav1.NamespaceAll).Required(false)).
			Produces(restful.MIME_JSON).
			Writes(fv1.Function{}). // on the response
			Returns(http.StatusOK, "A function", fv1.Function{}))

	ws.Route(
		ws.PUT("/v2/functions/{function}").
			Doc("Update function").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.PathParameter("function", "Function name").DataType("string").DefaultValue("").Required(true)).
			Produces(restful.MIME_JSON).
			Reads(fv1.Function{}).
			Writes(metav1.ObjectMeta{}). // on the response
			Returns(http.StatusOK, "ObjectMeta of updated function", metav1.ObjectMeta{}))

	ws.Route(
		ws.DELETE("/v2/functions/{function}").
			Doc("Delete function").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.PathParameter("function", "Function name").DataType("string").DefaultValue("").Required(true)).
			Param(ws.QueryParameter("namespace", "Namespace of function").DataType("string").DefaultValue(metav1.NamespaceAll).Required(false)).
			Produces(restful.MIME_JSON).
			Returns(http.StatusOK, "Only HTTP status returned", nil))

	ws.Route(
		ws.GET("/v2/functions/logs/{function}").
			Doc("Get function logs from pod directly").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.PathParameter("function", "Function name").DataType("string").DefaultValue("").Required(true)).
			Param(ws.QueryParameter("namespace", "Namespace of function").DataType("string").DefaultValue(metav1.NamespaceDefault).Required(false)).
			Produces(restful.MIME_JSON).
			Writes(restful.MIME_JSON). // on the response
			Returns(http.StatusOK, "Function Logs", logdb.Logs{}))
}

func (a *API) FunctionApiList(w http.ResponseWriter, r *http.Request) {
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	funcs, err := a.fissionClient.CoreV1().Functions(ns).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(funcs.Items)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) FunctionApiCreate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var f fv1.Function
	err = json.Unmarshal(body, &f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// check if namespace exists, if not create it.
	err = a.createNsIfNotExists(f.ObjectMeta.Namespace)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	fnew, err := a.fissionClient.CoreV1().Functions(f.ObjectMeta.Namespace).Create(&f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(fnew.ObjectMeta)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	a.respondWithSuccess(w, resp)
}

func (a *API) FunctionApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["function"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	f, err := a.fissionClient.CoreV1().Functions(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) FunctionApiUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["function"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var f fv1.Function
	err = json.Unmarshal(body, &f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	if name != f.ObjectMeta.Name {
		err = ferror.MakeError(ferror.ErrorInvalidArgument, "Function name doesn't match URL")
		a.respondWithError(w, err)
		return
	}

	fnew, err := a.fissionClient.CoreV1().Functions(f.ObjectMeta.Namespace).Update(&f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(fnew.ObjectMeta)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) FunctionApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["function"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.CoreV1().Functions(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}

// FunctionLogsApiPost establishes a proxy server to log database, and redirect
// query command send from client to database then proxy back the db response.
func (a *API) FunctionLogsApiPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// get dbType from url
	dbType := vars["dbType"]

	// find correspond db http url
	dbCnf := a.getLogDBConfig(dbType)

	svcUrl, err := url.Parse(dbCnf.httpURL)
	if err != nil {
		msg := "failed parse url to establish proxy to database for function logs"
		a.logger.Error(msg,
			zap.Error(err),
			zap.String("database_url", dbCnf.httpURL))
		a.respondWithError(w, errors.Wrap(err, msg))
		return
	}
	// set up proxy server director
	director := func(req *http.Request) {
		// only replace url Scheme and Host to remote influxDB
		// and leave query string intact
		req.URL.Scheme = svcUrl.Scheme
		req.URL.Host = svcUrl.Host
		req.URL.Path = svcUrl.Path
		req.Host = svcUrl.Host
		// set up http basic auth for database authentication
		req.SetBasicAuth(dbCnf.username, dbCnf.password)
	}
	proxy := &httputil.ReverseProxy{
		Director: director,
	}
	proxy.ServeHTTP(w, r)
}

// FunctionPodLogs : Get logs for a function directly from pod
func (a *API) FunctionPodLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fnName := vars["function"]

	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}
	// since time
	sinceTimeStr := a.extractQueryParamFromRequest(r, "stime")
	sinceTime, _ := strconv.ParseInt(sinceTimeStr, 10, 64)
	podNs := "fission-function"

	f, err := a.fissionClient.CoreV1().Functions(ns).Get(fnName, metav1.GetOptions{})
	if err != nil {
		a.respondWithError(w, errors.New(fmt.Sprintf("dont have information of this function %v in namespace %v", fnName, ns)))
		return
	}

	// Get function Pods first
	selector := map[string]string{
		fv1.FUNCTION_UID:          string(f.ObjectMeta.UID),
		fv1.ENVIRONMENT_NAME:      f.Spec.Environment.Name,
		fv1.ENVIRONMENT_NAMESPACE: f.Spec.Environment.Namespace,
	}

	if f.Spec.Environment.Namespace != metav1.NamespaceDefault {
		podNs = f.Spec.Environment.Namespace
	}

	podList, err := a.kubernetesClient.CoreV1().Pods(podNs).List(metav1.ListOptions{
		LabelSelector: labels.Set(selector).AsSelector().String(),
	})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// Get the logs for last Pod executed
	pods := podList.Items
	sort.Slice(pods, func(i, j int) bool {
		rv1, _ := strconv.ParseInt(pods[i].ObjectMeta.ResourceVersion, 10, 32)
		rv2, _ := strconv.ParseInt(pods[j].ObjectMeta.ResourceVersion, 10, 32)
		return rv1 > rv2
	})

	if len(pods) <= 0 {
		_, _ = w.Write(createErrorResponseJsonString("no active pod found"))
		return
	}
	snt := time.Unix(sinceTime/1e9, sinceTime%1e9)
	sn := metav1.NewTime(snt)
	// get the pod with highest resource version
	podLogOpts := apiv1.PodLogOptions{
		Container:  f.Spec.Environment.Name,
		SinceTime:  &sn,
		Timestamps: true,
		Follow:     false,
	}
	err = getContainerLog(a.kubernetesClient, w, &pods[0], podLogOpts)
	if err != nil {
		a.respondWithError(w, errors.Wrapf(err, "error getting container logs"))
		return
	}
}

func getContainerLog(kubernetesClient *kubernetes.Clientset, w http.ResponseWriter, pod *apiv1.Pod, podLogOpts apiv1.PodLogOptions) error {
	podLogsReq := kubernetesClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.ObjectMeta.Name, &podLogOpts)

	podLogs, err := podLogsReq.Stream()
	if err != nil {
		return errors.Wrapf(err, "error streaming pod log")
	}

	logs, err := ioutil.ReadAll(podLogs)
	if err != nil {
		return errors.Wrapf(err, "error copying pod log")
	}

	msgMap := map[string]string{
		"status":    "success",
		"Pod":       pod.Name,
		"logs":      string(logs),
		"timestamp": "",
	}

	msg, err := json.Marshal(msgMap)
	_, _ = w.Write(msg)
	podLogs.Close()
	return nil
}

func createErrorResponseJsonString(msg string) []byte {
	response := map[string]string{
		"status": "error",
		"msg":    msg,
	}
	rep, err := json.Marshal(response)
	if err != nil {
		return []byte("{\"status\":\"down\"}")
	}
	return rep
}
