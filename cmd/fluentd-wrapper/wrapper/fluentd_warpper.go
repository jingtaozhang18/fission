package wrapper

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var (
	fluentd *exec.Cmd
)

func StartFluentd(zapLogger *zap.Logger) error {
	if fluentd != nil {
		return errors.New("fluentd already started")
	}
	zapLogger.Info("start fluentd")
	fluentd = exec.Command("fluentd", "-c", "/fluentd/etc/fluent.conf")
	fluentd.Stderr = os.Stderr
	fluentd.Stdout = os.Stdout
	return fluentd.Start()
}

func shell(command string) string {
	cmd := exec.Command("/bin/sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("error %v", err)
	}
	return string(out)
}

var loadCh = make(chan int, 1)

func ReloadFluentd() error {
	if fluentd == nil {
		return fmt.Errorf("fluentd have not started")
	}
	select {
	case loadCh <- 1:
		go func() {
			log.Printf("reload fluentd %v", fluentd.Process.Pid)
			command := fmt.Sprintf("pgrep -P %d", fluentd.Process.Pid)
			childID := shell(command)
			log.Printf("before reload childId : %s", childID)
			fluentd.Process.Signal(syscall.SIGUSR2)
			time.Sleep(5 * time.Second)
			afterChildID := shell(command)
			log.Printf("after reload childId : %s", afterChildID)
			if childID == afterChildID {
				log.Printf("kill childId : %s", childID)
				shell("kill -9 " + childID)
			}
			<-loadCh
		}()
	default:
		return fmt.Errorf("fluentd is reloading")
	}
	return nil
}
