package logger

import (
	"bytes"
	"fmt"
	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/cache"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/publisher"
	"github.com/fission/fission/pkg/timercheck"
	"go.uber.org/zap"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"
	"os"
	"strings"
	"text/template"
	"time"
)

const (
	configsPathTemplate    = "/fluentd/etc/config.d/%s"
	fluentdPosPathTemplate = "/fluentd/pos/pos__%s_%s.pos"
	configTemplate         = `<source>
    @type tail
    path %s
    pos_file %s
    read_from_head true
    emit_unmatched_lines true
	refresh_interval 20
    tag %s
    <parse>
        @type json
    </parse>
</source>

<filter %s>
    @type record_transformer             
    <record>
        tag ${tag}
    </record>
</filter>

<match %s>
	@type copy
	%s
</match>
`
)

type FluentdGen struct {
	configRelatedFunction *cache.Cache
	timer                 *timercheck.TimerCheck
	zapLogger             *zap.Logger
	k8sClientSet          *kubernetes.Clientset
	fissionClient         *crd.FissionClient
	poster                publisher.Publisher
}

func makeFluentdGen(zapLogger *zap.Logger, k8sClientSet *kubernetes.Clientset, fissionClient *crd.FissionClient) (*FluentdGen, error) {
	poster := publisher.MakeWebhookPublisher(zapLogger, "http://127.0.0.1:8090")
	timer := timercheck.MakeTimerChecker(
		time.Second*4,
		func() {
			poster.Publish("", map[string]string{}, "/update")
			zapLogger.Info("call fluentd to update configs")
		},
	)
	return &FluentdGen{
		configRelatedFunction: cache.MakeCache(0, 0),
		timer:                 timer,
		zapLogger:             zapLogger,
		k8sClientSet:          k8sClientSet,
		fissionClient:         fissionClient,
		poster:                poster,
	}, nil
}

func formatConfig(tpl string, function *fv1.Function) (string, error) {
	tmpl, err := template.New("xx").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, function)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (fg *FluentdGen) server() {
	// run controller
	funcController := fg.makeFunctionChangeController()
	cfmController := fg.makeConfigChangeController()
	go funcController.Run(make(chan struct{}))
	go cfmController.Run(make(chan struct{}))
	go fg.timer.DoCircle()
}

func (fg *FluentdGen) genFluentdConfigByFunction(function *fv1.Function) {
	logConfigs, _ := fg.getRelatedConfigs(function)
	fg.writeFluentdConfig(function, logConfigs)
}

func (fg *FluentdGen) genFluentdConfigByConfig(cfm *v1.ConfigMap) {
	functions, err := fg.getRelatedFunction(cfm)
	if err != nil {
		fg.zapLogger.Error(fmt.Sprintf("get related function error: %s", err))
		return
	}
	for _, function := range functions {
		fg.genFluentdConfigByFunction(&function)
	}

}

func (fg *FluentdGen) writeFluentdConfig(function *fv1.Function, logConfigs []string) {
	logPath := fmt.Sprintf("%s/%s_%s_*", fissionSymlinkPath, function.ObjectMeta.Namespace, function.ObjectMeta.Name)
	posPath := fmt.Sprintf(fluentdPosPathTemplate, function.ObjectMeta.Namespace, function.ObjectMeta.Name)
	tag := fmt.Sprintf("%s.%s", function.ObjectMeta.Namespace, function.ObjectMeta.Name)
	// logConfigs can have some variable, so it need to construct.
	var formatLogConfigs []string
	for _, lc := range logConfigs {
		flc, err := formatConfig(lc, function)
		if err != nil {
			fg.zapLogger.Error(fmt.Sprintf("format config error: %v, functionNamespace: %s, function:%s template: %s", err, function.ObjectMeta.Namespace, function.ObjectMeta.Name, lc))
		}
		formatLogConfigs = append(formatLogConfigs, flc)
	}
	output := strings.Join(formatLogConfigs, "\n")
	configContent := fmt.Sprintf(configTemplate, logPath, posPath, tag, tag, tag, output)
	funcConfigPath := fmt.Sprintf(configsPathTemplate, function.ObjectMeta.Namespace+"_"+function.ObjectMeta.Name+".conf")
	fg.zapLogger.Debug(fmt.Sprintf("%s %s update config", function.ObjectMeta.Namespace, function.ObjectMeta.Name))
	// 直接写入
	fg.timer.Update(func() {
		if len(formatLogConfigs) == 0 {
			// delete the file if it exists
			err := os.Remove(funcConfigPath)
			if err != nil {
				fg.zapLogger.Warn(fmt.Sprintf("delete %s failed, error: %v", funcConfigPath, err))
			}
		} else {
			err := ioutil.WriteFile(funcConfigPath, []byte(configContent), os.FileMode(0644))
			if err != nil {
				fg.zapLogger.Error(fmt.Sprintf("write %s file failed, error: %v", funcConfigPath, err))
			}
		}
	})
}

func (fg *FluentdGen) getRelatedConfigs(function *fv1.Function) ([]string, error) {
	// only to use the special config with the function
	functionConfigMapName := fmt.Sprintf(fv1.LogConfigMapName, function.ObjectMeta.Name)
	functionConfigMapExist := false
	for _, cfm := range function.Spec.ConfigMaps {
		if cfm.Namespace == function.ObjectMeta.Namespace && cfm.Name == functionConfigMapName {
			functionConfigMapExist = true
			break
		}
	}
	if !functionConfigMapExist {
		fg.zapLogger.Warn(fmt.Sprintf("Namespace:%s, Function:%s dont carry the special configMap:%s", function.ObjectMeta.Namespace, function.ObjectMeta.Name, fv1.LogConfigMapName))
		return []string{}, nil
	}
	functionConfigMap, err := fg.k8sClientSet.CoreV1().ConfigMaps(function.ObjectMeta.Namespace).Get(functionConfigMapName, metav1.GetOptions{})
	if err != nil {
		fg.zapLogger.Error(fmt.Sprintf("Namespace:%s, Function:%s get configMap %s error, ", function.ObjectMeta.Namespace, function.ObjectMeta.Name, functionConfigMapName))
		return []string{}, err
	}
	if logTypes, ok := functionConfigMap.Data[fv1.LogCollectionConfigKey]; ok {
		globalLogConfigMap, gErr := fg.k8sClientSet.CoreV1().ConfigMaps(fv1.GlobalSecretConfigMapNS).Get(fv1.LogCollectionConfigName, metav1.GetOptions{})
		localLogConfigMap, lErr := fg.k8sClientSet.CoreV1().ConfigMaps(function.ObjectMeta.Namespace).Get(fv1.LogCollectionConfigName, metav1.GetOptions{})
		// if dont get the config, only log, do nothing
		if gErr != nil {
			fg.zapLogger.Warn(fmt.Sprintf("Namespace:%s, ConfigMap:%s dont have global log config.", fv1.GlobalSecretConfigMapNS, fv1.LogCollectionConfigName))
		}
		if lErr != nil {
			fg.zapLogger.Warn(fmt.Sprintf("Namespace:%s, ConfigMap:%s dont have local log config.", function.ObjectMeta.Namespace, fv1.LogCollectionConfigName))
		}
		var logConfigs []string
		var foundFlag bool
		for _, logType := range strings.Split(logTypes, ",") {
			foundFlag = false
			if lErr == nil {
				if logConfig, ok := localLogConfigMap.Data[logType]; ok {
					logConfigs = append(logConfigs, logConfig)
					foundFlag = true
				}
			} else if gErr == nil {
				if logConfig, ok := globalLogConfigMap.Data[logType]; ok {
					logConfigs = append(logConfigs, logConfig)
					foundFlag = true
				}
			}
			if foundFlag == false {
				fg.zapLogger.Warn(fmt.Sprintf("Namespace:%v, Function:%v, LogType:%v not found config", function.ObjectMeta.Namespace, function.ObjectMeta.Name, logType))
			}
		}
		return logConfigs, nil
	}
	return []string{}, nil
}
func (fg *FluentdGen) getRelatedFunction(m *v1.ConfigMap) ([]fv1.Function, error) {
	funcList, err := fg.fissionClient.CoreV1().Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return []fv1.Function{}, err
	}
	// almost the same method of executor/cms/cmscontroller
	relatedFunctions := make([]fv1.Function, 0)
	if m.Name == fv1.LogCollectionConfigName {
		for _, f := range funcList.Items {
			if m.Namespace == fv1.GlobalSecretConfigMapNS || f.Namespace == m.Namespace {
				relatedFunctions = append(relatedFunctions, f)
			}
		}
	} else {
		for _, f := range funcList.Items {
			for _, cm := range f.Spec.ConfigMaps {
				if (cm.Name == m.Name) && (cm.Namespace == m.Namespace) {
					relatedFunctions = append(relatedFunctions, f)
					break
				}
			}
		}
	}
	return relatedFunctions, nil
}
func (fg *FluentdGen) makeFunctionChangeController() k8sCache.Controller {
	resyncPeriod := 2 * time.Second
	lw := k8sCache.NewListWatchFromClient(fg.fissionClient.CoreV1().RESTClient(), "functions", metav1.NamespaceAll, fields.Everything())
	_, controller := k8sCache.NewInformer(lw, &fv1.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				function := obj.(*fv1.Function)
				fg.genFluentdConfigByFunction(function)
			},
			UpdateFunc: func(oldObj interface{}, obj interface{}) {
				oldFunction := oldObj.(*fv1.Function)
				function := obj.(*fv1.Function)
				if fmt.Sprintf("%v", oldFunction) == fmt.Sprintf("%v", function) {
					return
				}
				fg.genFluentdConfigByFunction(function)
			},
			DeleteFunc: func(obj interface{}) {
				// todo do nothing or delete the config
				//function := obj.(*fv1.Function)
			},
		})
	return controller
}

func (fg *FluentdGen) makeConfigChangeController() k8sCache.Controller {
	resyncPeriod := 2 * time.Second
	lw := k8sCache.NewListWatchFromClient(fg.k8sClientSet.CoreV1().RESTClient(), "configmaps", metav1.NamespaceAll, fields.Everything())
	_, controller := k8sCache.NewInformer(lw, &v1.ConfigMap{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cfm := obj.(*v1.ConfigMap)
				fg.genFluentdConfigByConfig(cfm)
			},
			UpdateFunc: func(oldObj interface{}, obj interface{}) {
				oldCfm := oldObj.(*v1.ConfigMap)
				cfm := obj.(*v1.ConfigMap)
				if oldCfm.ObjectMeta.ResourceVersion == cfm.ObjectMeta.ResourceVersion {
					return
				}
				fg.genFluentdConfigByConfig(cfm)
			},
			DeleteFunc: func(obj interface{}) {
				// todo do nothing or delete the config
				//cfm := obj.(*v1.ConfigMap)
			},
		})
	return controller
}
