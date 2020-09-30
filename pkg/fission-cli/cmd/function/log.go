/*
Copyright 2019 The Fission Authors.

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

package function

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/pkg/fission-cli/cliwrapper/cli"
	"github.com/fission/fission/pkg/fission-cli/cmd"
	flagkey "github.com/fission/fission/pkg/fission-cli/flag/key"
	"github.com/fission/fission/pkg/fission-cli/logdb"
	"github.com/fission/fission/pkg/fission-cli/util"
)

type LogSubCommand struct {
	cmd.CommandActioner
}

func Log(input cli.Input) error {
	return (&LogSubCommand{}).do(input)
}

func (opts *LogSubCommand) do(input cli.Input) error {
	dbType := input.String(flagkey.FnLogDBType)
	fnPod := input.String(flagkey.FnLogPod)
	kubeContext := input.String(flagkey.KubeContext)

	logReverseQuery := !input.Bool(flagkey.FnLogFollow) && input.Bool(flagkey.FnLogReverseQuery)

	recordLimit := input.Int(flagkey.FnLogCount)
	if recordLimit <= 0 {
		recordLimit = 1000
	}

	f, err := opts.Client().V1().Function().Get(&metav1.ObjectMeta{
		Name:      input.String(flagkey.FnName),
		Namespace: input.String(flagkey.NamespaceFunction),
	})
	if err != nil {
		return errors.Wrap(err, "error getting function")
	}

	server, err := util.GetApplicationUrl("application=fission-api", kubeContext)
	if err != nil {
		return err
	}

	// request the controller to establish a proxy server to the database.
	logDB, err := logdb.GetLogDB(dbType, server)
	if err != nil {
		return errors.Wrapf(err, "failed to get log database")
	}

	requestChan := make(chan struct{})
	responseChan := make(chan struct{})
	ctx := context.Background()

	go func(ctx context.Context, requestChan, responseChan chan struct{}) {
		// from now to output log
		var t time.Time
		if input.Bool(flagkey.FnLogFromNow) {
			t = time.Now()
		} else {
			t = time.Unix(0, 0*int64(time.Millisecond))
		}
		podName := "none"
		var cstSh, _ = time.LoadLocation("Asia/Shanghai") //上海
		for {
			select {
			case <-requestChan:
				logFilter := logdb.LogFilter{
					Pod:               fnPod,
					Function:          f.ObjectMeta.Name,
					FunctionNamespace: f.ObjectMeta.Namespace,
					FuncUid:           string(f.ObjectMeta.UID),
					Since:             t,
					Reverse:           logReverseQuery,
					RecordLimit:       recordLimit,
				}
				logEntries, err := logDB.GetLogs(logFilter)
				if err != nil {
					fmt.Printf("Error querying logs: %v", err)
					responseChan <- struct{}{}
					return
				}
				for _, logEntry := range logEntries {
					t = logEntry.Timestamp
					if podName != logEntry.Pod {
						podName = logEntry.Pod
						fmt.Printf("\n**** logs from %v ****\n\n", podName)
					}
					if input.Bool(flagkey.FnLogDetail) {
						if input.Bool(flagkey.FnLogWithTime) {
							fmt.Printf("Timestamp: %s\nNamespace: %s\nFunction Name: %s\nFunction ID: %s\nPod: %s\nContainer: %s\nStream: %s\nLog: %s\n---\n",
								logEntry.Timestamp.In(cstSh).Format("2006-01-02 15:04:05"), logEntry.Namespace, logEntry.FuncName, logEntry.FuncUid, logEntry.Pod, logEntry.Container, logEntry.Stream, logEntry.Message)
						} else {
							fmt.Printf("Namespace: %s\nFunction Name: %s\nFunction ID: %s\nPod: %s\nContainer: %s\nStream: %s\nLog: %s\n---\n",
								logEntry.Namespace, logEntry.FuncName, logEntry.FuncUid, logEntry.Pod, logEntry.Container, logEntry.Stream, logEntry.Message)
						}
					} else {
						if input.Bool(flagkey.FnLogWithTime) {
							fmt.Printf("[%s] %s\n", logEntry.Timestamp.In(cstSh).Format("2006-01-02 15:04:05"), logEntry.Message)
						} else {
							fmt.Printf("%s\n", logEntry.Message)
						}
					}
				}
				responseChan <- struct{}{}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, requestChan, responseChan)

	for {
		requestChan <- struct{}{}
		time.Sleep(1 * time.Second)

		<-responseChan
		if !input.Bool(flagkey.FnLogFollow) {
			ctx.Done()
			break
		}
	}

	return nil
}
