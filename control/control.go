package control

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"gamepctl/config"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// Control 游戏操作控制的结构体
type Control struct {
	Logger      log.Logger
	Cmdcfg      *config.CmdConfig
	Failed      []FailedData              // 执行command时，未成功的记录
	Offline     []FailedData              // start后，校验时未成功开启的记录
	StillOnline map[string]map[string]int // stop后，校验时仍然在线的记录(stop失败)
	Lock        *sync.Mutex
}

type FailedData struct {
	name    string
	command string
	port    int
}

var (
	onLineProData map[string]int = make(map[string]int)
	//offLine       []FailedData
	onLineSerData map[string]string = make(map[string]string)
)

func NewControl(logger log.Logger, cfg *config.CmdConfig, lock *sync.Mutex) *Control {
	return &Control{
		Logger: logger,
		Cmdcfg: cfg,
		Lock:   lock,
	}
}
func (c *Control) Run() {
	failed := FailedData{}
	c.Offline = []FailedData{}
	wg := sync.WaitGroup{}
	//wg2 := sync.WaitGroup{}
	// switch c.Cmdcfg.Kind {
	// case "start":
	fmt.Printf("[x]%s [%s] game process\n", c.Cmdcfg.Kind, c.Cmdcfg.Metadata.Annotations)
	for _, preJob := range c.Cmdcfg.Spec.PreExec {
		status := "Success"
		if strings.ToLower(preJob.Type) == "ssh" {
			//ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			//defer cancel()
			shell := []string{fmt.Sprintf("-p%s", c.Cmdcfg.Metadata.Port), fmt.Sprintf("%s@%s", c.Cmdcfg.Metadata.User, preJob.Server)}
			shell = append(shell, preJob.Command...)
			//fmt.Println("shell = ", shell)
			cmd := exec.Command("ssh", shell...)
			var stdErr bytes.Buffer
			var stdOut bytes.Buffer
			cmd.Stderr = &stdErr
			cmd.Stdout = &stdOut
			if err := cmd.Run(); err != nil {
				level.Error(c.Logger).Log("Error", err, "StdErr", stdErr.String())
				status = "Failed"
				failed.name = preJob.Server
				failed.command = strings.Join(preJob.Command, " ")
				c.Failed = append(c.Failed, failed)
			}
			fmt.Printf("Server:%s Type:%s Command:%s [%s]\n", preJob.Server, preJob.Type, preJob.Command, status)
			time.Sleep(time.Duration(preJob.Wait) * time.Second)
		} else if strings.ToLower(preJob.Type) == "http" {
			client := newHttpClient()
			resp, err := client.Get(preJob.Url)
			if err != nil {
				level.Error(c.Logger).Log("Error", err.Error())
				status = "Failed"
				failed.name = preJob.Server
				failed.command = preJob.Url
				c.Failed = append(c.Failed, failed)
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				level.Error(c.Logger).Log("StatusCode", resp.StatusCode)
				status = "Failed"
				failed.name = preJob.Server
				failed.command = preJob.Url
				c.Failed = append(c.Failed, failed)
			}
			fmt.Printf("Server:%s Type:%s Url:%s [%s]", preJob.Server, preJob.Type, preJob.Url, status)
			time.Sleep(time.Duration(preJob.Wait) * time.Second)
		}
	}
	// 批量执行
	ch := make(chan struct{}, c.Cmdcfg.Metadata.Concurrence)
	defer close(ch)
	ch2 := make(chan struct{}, c.Cmdcfg.Metadata.Concurrence)
	defer close(ch2)
	for _, job := range c.Cmdcfg.Spec.Exec {
		wg.Add(1)
		go concurrentExec(job, c.Cmdcfg.Metadata, c.Logger, &c.Failed, c.Lock, &wg, ch)
	}
	wg.Wait()
	// start stop 校验方式区分
	fmt.Printf("[x]check [%s] game process\n", c.Cmdcfg.Metadata.Annotations)
	time.Sleep(1 * time.Second)
	switch c.Cmdcfg.Kind {
	case "start":
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cmdcfg.Metadata.Timeout)*time.Second)
		defer cancel()
	startCheckLoop:
		for {
			// wait 10s every loop
			time.Sleep(10 * time.Second)
			select {
			case <-ctx.Done():
				fmt.Println("stopped checking process,beacuse of timeout")
				break startCheckLoop
			default:
				var wg2 sync.WaitGroup
				for _, check := range c.Cmdcfg.Check.Data {
					wg2.Add(1)
					go concurrentCheck(c.Cmdcfg.Kind, check, c.Cmdcfg.Metadata, c.Logger, &c.Offline, c.Lock, &wg2, ch2)
				}
				wg2.Wait()
				// 如果没有不在线的进程了，那么就意味着全部在线啦,可以结束循环
				//fmt.Println("c.Online = ", c.Offline)
				if len(c.Offline) == 0 {
					break startCheckLoop
				}
			}

		}
	case "stop":
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cmdcfg.Metadata.Timeout)*time.Second)
		defer cancel()
	checkLoop:
		for {
			// wait 10s every loop
			time.Sleep(10 * time.Second)
			select {
			case <-ctx.Done():
				fmt.Println("stopped checking process,beacuse of timeout")
				break checkLoop
			default:
				var wg sync.WaitGroup
				for _, check := range c.Cmdcfg.Check.Data {
					wg.Add(1)
					go concurrentCheck(c.Cmdcfg.Kind, check, c.Cmdcfg.Metadata, c.Logger, &c.Offline, c.Lock, &wg, ch2)
				}
				wg.Wait()
				// 如果此时onLineSerData为空，证明没有进程在线了，可以退出循环
				if len(onLineSerData) == 0 {
					break checkLoop
				}
			}
		}
	}
	c.Offline = unique(c.Offline)
	fmt.Printf("[x]result of %s [%s] game process\n", c.Cmdcfg.Kind, c.Cmdcfg.Metadata.Annotations)
	if strings.ToLower(c.Cmdcfg.Kind) == "start" {
		if len(c.Offline) != 0 {
			if len(c.Failed) != 0 {
				for _, off := range c.Offline {
					fmt.Printf("Server:%s Process:%s [Offline] \n", off.name, off.command)
				}
				for _, fail := range c.Failed {
					fmt.Printf("Server:%s Command:%s [Failed] \n", fail.name, fail.command)
				}
				fmt.Println("start game process failed")
				os.Exit(-1)
			} else {
				for _, off := range c.Offline {
					fmt.Printf("Server:%s Process:%s [Offline] \n", off.name, off.command)
				}
				fmt.Println("start game process failed")
				os.Exit(-1)
			}

		} else {
			if len(c.Failed) != 0 {
				for _, fail := range c.Failed {
					fmt.Printf("Server:%s Command:%s [Failed] \n", fail.name, fail.command)
				}
				fmt.Println("start game process failed")
				os.Exit(-1)
			}
			fmt.Println("start game process success")
		}
	} else {
		if len(c.Failed) != 0 {
			for _, fail := range c.Failed {
				fmt.Printf("Server:%s Command:%s [Failed] \n", fail.name, fail.command)
			}
			fmt.Println("stop game process failed")
			os.Exit(-1)
		}
		if len(onLineSerData) != 0 {
			for k, v := range onLineSerData {
				fmt.Printf("Server:%s Process:%s [Online]\n", k, v)
			}
			fmt.Println("stop game process failed")
			os.Exit(-1)
		} else {
			fmt.Println("stop game process success")
		}
	}

}

func newHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// 不校验https证书
				InsecureSkipVerify: true,
			},
			MaxConnsPerHost:     300,
			MaxIdleConns:        150,
			MaxIdleConnsPerHost: 75,
			IdleConnTimeout:     10 * time.Second,
		},
	}
}

func concurrentExec(s config.Settings, m config.Metadata, logger log.Logger, failed *[]FailedData, lock *sync.Mutex, wg *sync.WaitGroup, ch chan struct{}) {
	defer wg.Done()
	defer func() { <-ch }()
	ch <- struct{}{}
	var errorData FailedData
	status := "Success"
	if strings.ToLower(s.Type) == "ssh" {
		shell := []string{fmt.Sprintf("-p%s", m.Port), fmt.Sprintf("%s@%s", m.User, s.Server)}
		shell = append(shell, s.Command...)
		cmd := exec.Command("ssh", shell...)
		var stdErr bytes.Buffer
		var stdOut bytes.Buffer
		cmd.Stderr = &stdErr
		cmd.Stdout = &stdOut
		if err := cmd.Run(); err != nil {
			level.Error(logger).Log("Error", err, "StdErr", stdErr.String())
			status = "Failed"
			errorData.name = s.Server
			errorData.command = strings.Join(s.Command, " ")
			lock.Lock()
			*failed = append(*failed, errorData)
			lock.Unlock()
		}
		fmt.Printf("Server:%s Type:%s Command:%s [%s]\n", s.Server, s.Type, s.Command, status)
	} else if strings.ToLower(s.Type) == "http" {
		client := newHttpClient()
		resp, err := client.Get(s.Url)
		if err != nil {
			level.Error(logger).Log("Error", err.Error())
			status = "Failed"
			errorData.name = s.Server
			errorData.command = s.Url
			lock.Lock()
			*failed = append(*failed, errorData)
			lock.Unlock()
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			level.Error(logger).Log("StatusCode", resp.StatusCode)
			status = "Failed"
			errorData.name = s.Server
			errorData.command = s.Url
			lock.Lock()
			*failed = append(*failed, errorData)
			lock.Unlock()
		}
		fmt.Printf("Server:%s Type:%s Url:%s [%s]\n", s.Server, s.Type, s.Url, status)
	}
}

func concurrentCheck(k string, s config.Settings, m config.Metadata, logger log.Logger, offLine *[]FailedData, lock *sync.Mutex, wg *sync.WaitGroup, ch chan struct{}) {
	defer wg.Done()
	defer func() { <-ch }()
	var offData FailedData
	ch <- struct{}{}
	if strings.ToLower(s.Type) == "ps" {
		shell := []string{
			fmt.Sprintf("-p%s", m.Port),
			fmt.Sprintf("%s@%s", m.User, s.Server),
			fmt.Sprintf("ps -ef|grep \"%s\"|egrep -v \"grep|SCREEN\"|wc -l", s.Process),
		}
		cmd := exec.Command("ssh", shell...)
		var stdErr bytes.Buffer
		var stdOut bytes.Buffer
		cmd.Stderr = &stdErr
		cmd.Stdout = &stdOut
		if err := cmd.Run(); err != nil {
			level.Error(logger).Log("Error", err, "StdErr", stdErr.String())
			offData.name = s.Server
			offData.command = s.Process
			lock.Lock()
			*offLine = append(*offLine, offData)
			lock.Unlock()
			return
		}
		n, err := strconv.Atoi(strings.Trim(stdOut.String(), "\n"))
		if err != nil {
			level.Error(logger).Log("Error", err)
			offData.name = s.Server
			offData.command = s.Process
			lock.Lock()
			*offLine = append(*offLine, offData)
			lock.Unlock()
			return
		}
		fmt.Printf("Server:%s Process:%s ProcessNumber: %d [checked]\n", s.Server, s.Process, n)
		switch k {
		case "start":
			if n != s.Number {
				offData.name = s.Server
				offData.command = s.Process
				lock.Lock()
				*offLine = append(*offLine, offData)
				lock.Unlock()
			} else {
				lock.Lock()
				onLineProData[s.Server+"_"+s.Process] = n
				onLineSerData[s.Server] = s.Process
				lock.Unlock()
			}
		case "stop":
			//fmt.Println(" 接收到了", k, n)
			if n != s.Number {
				lock.Lock()
				onLineProData[s.Server+"_"+s.Process] = n
				onLineSerData[s.Server] = s.Process
				lock.Unlock()
			} else {
				if _, ok := onLineProData[s.Server+"_"+s.Process]; ok {
					lock.Lock()
					delete(onLineProData, s.Server+"_"+s.Process)
					delete(onLineSerData, s.Server)
					lock.Unlock()
				}
			}
		}

	} else if strings.ToLower(s.Type) == "port" {
		for _, port := range s.Ports {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.Server, port), 5*time.Second)
			if err != nil {
				fmt.Printf("Server:%s Process:%s Port: %d [%s]\n", s.Server, s.Process, port, err)
				switch k {
				case "start":
					// 如果是start操作，那么这些都是未成功开启的端口
					offData.name = s.Server
					offData.command = s.Process + ":" + strconv.Itoa(port)
					offData.port = port
					lock.Lock()
					*offLine = append(*offLine, offData)
					lock.Unlock()
				case "stop":
					// 如果是stop操作，那么这些都是成功关闭的
					// 在循环查询时，需要删除之前没来得及关闭但是现在已经关闭的端口的信息
					if _, ok := onLineProData[s.Server+"_"+s.Process+":"+strconv.Itoa(port)]; ok {
						lock.Lock()
						delete(onLineProData, s.Server+"_"+s.Process+":"+strconv.Itoa(port))
						delete(onLineSerData, s.Server+"["+strconv.Itoa(port)+"]")
						lock.Unlock()
					}
				}
				continue
			}
			defer conn.Close()
			// 此处都是依旧在线的端口
			fmt.Printf("Server:%s Process:%s Port: %d [checked]\n", s.Server, s.Process, port)
			switch k {
			case "start":
				lock.Lock()
				onLineProData[s.Server+"_"+s.Process+":"+strconv.Itoa(port)] = 1
				onLineSerData[s.Server+"["+strconv.Itoa(port)+"]"] = s.Process + ":" + strconv.Itoa(port)
				// 如果是start操作，那么这些端口都是符合预期的,要删除之前因为启动慢而导致端口还未监听的失败端口
				for i := 0; i < len(*offLine); i++ {
					if (*offLine)[i].port == port {
						*offLine = append((*offLine)[:i], (*offLine)[i+1:]...)
						i--
					}
				}
				lock.Unlock()
			case "stop":
				// 如果是stop操作，那么这些都是不合服预期的，需要保存状态
				lock.Lock()
				onLineProData[s.Server+"_"+s.Process+":"+strconv.Itoa(port)] = 1
				onLineSerData[s.Server+"["+strconv.Itoa(port)+"]"] = s.Process + "->Port:" + strconv.Itoa(port)
				lock.Unlock()
			}
		}
	}
}

// 去重
func unique(slice []FailedData) []FailedData {
	seen := make(map[FailedData]struct{})
	result := make([]FailedData, 0, len(slice))

	for _, value := range slice {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
