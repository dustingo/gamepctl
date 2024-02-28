### Gamepctl

gamectl是用于控制对游戏的业务运维操作(开、关)的控制工具。
同服务器端更新工具一样，游戏业务的日常开关运维操作本质上也是"shell"或者API的集合，因此gamectl也同样采取了通过对配置的解析实现操作的进行并尽量避免重复的开发、编译工作。

### 配置文件
- 配置文件命名
gamectl当前支持开、关两种操作，为了方便区分，配置文件强制以"start"开头来表示"开"操作，以"stop"开头来表示"关"操作。正因此强制设定，你也可以通过命名多个配置文件的方式来实现对多组服务器的控制，如"start_s1.yaml"、"start_s2.yaml"。
- 配置文件解析
配置文件总共分为4部分：
1. kind： string;表示当前配置文件内实际表示的操作,可选有：【start|stop】
2. metadata：操作的一些全局设置  
   - user:  string;服务器用户，如：root，server等
   - port:  string;服务器SSH端口，如"22"
   - timeout: 执行操作时的超时时间，单位s
   - concurrence: int；执行"exec"和"check"时的最大并发量
   - annotations: string;主要用于对此配置文件或操作对象的一个声明或标识,如"日服"
3. spec： 对"操作"的定义
    - preExec： 定义某些进程开、关顺序，此处会按顺序执行
      - server: string; 服务器的IP地址或hostname
      - type: string; 执行方式，可选: 【local|remote】
      - wait: int; 执行完后等待的时间，单位为second
      - command: []string；type为ssh时，要执行的命令，如:  ["cd /tmp","&&", "date >>time.date"]

    - exec: 对于除了需要指定操作顺序的进程之外，定义其他所有没有操作顺序的进程操作。字段类型同preExec，执行速度受metadata.concurrence限制

4. check: preExec与exec执行完后最终的校验部分定义
   - type:  string; 执行方式，可选: 【local|remote|port】
   - server:   string; 服务器的IP地址或hostname
   - process: string;当type=ps时使用，进程的唯一标识，通过ps检查此唯一标识确认后台进程是否启动
   - num: int;当type=ssh时使用，process标识的进程预期数量，如开启的时候我们预期数量为1，关闭的时候预期数量为0
   - port: []int； 当type=port时使用，

### 关于Type=http
目前来讲,只有在存在守护进程时，启动或关闭调用http接口时才有效。在进行后续检查时由于定制性太差，删除了检查的http方法。
### 使用
```shell
# 当配置文件与执行操作不符时
[root@adver test]# ./gamepctl apply -f start.yaml stop
Error: config file should start with "stop"

#当配置文件kind与操作不符时
[root@adver test]# ./gamepctl apply -f start.yaml start
Error: running start operation but kind is  stop

# 执行start操作
[root@adver test]# ./gamepctl apply -f start.yaml start
[x]start [国服] game process
Server:game92 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game91 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game89 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game90 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game87 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game88 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
[x]check [国服] game process
Server:game97 Process:hostwatch ProcessNumber: 1 [checked]
Server:game92 Process:hostwatch ProcessNumber: 1 [checked]
[x]result of start [国服] game process
start game process success

# 执行start操作时有错误时
[root@adver test]# ./gamepctl apply -f start.yaml start
[x]start [国服] game process
level=error caller=control.go:69 Error="exit status 127" StdErr="bash: dates: command not found\n"
Server:game92 Type:ssh Command:[cd /tmp && dates >>time.date] [Failed]
Server:game91 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
level=error caller=control.go:219 Error="exit status 255" StdErr="ssh: Could not resolve hostname game901: Name or service not known\r\n"
Server:game901 Type:ssh Command:[cd /tmp && date >> time.date] [Failed]
Server:game89 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game87 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game88 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
[x]check [国服] game process
Server:game97 Process:hostwatch ProcessNumber: 1 [checked]
Server:game92 Process:hostwatch ProcessNumber: 1 [checked]
[x]result of start [国服] game process
Server:game92 Command:cd /tmp && dates >>time.date [Failed] 
Server:game901 Command:cd /tmp && date >> time.date [Failed] 
start game process failed

# 执行关闭操作时,进程hostwatch预期为0，但实际上没有被关闭时
[root@adver test]# ./gamepctl apply -f stop.yaml stop
[x]stop [国服] game process
Server:game92 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game91 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game90 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game89 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game88 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game87 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
[x]check [国服] game process
Server:game97 Process:hostwatch ProcessNumber: 1 [checked]
Server:game92 Process:hostwatch ProcessNumber: 1 [checked]
stopped checking process,beacuse of timeout
[x]result of start [国服] game process
Server:game97 Process:hostwatch [Online]
Server:game92 Process:hostwatch [Online]
stop game process failed

# 执行关闭后，校验时进程数量以及端口符合预期时
[root@adver test]# ./gamepctl apply -f stop.yaml stop
[x]stop [国服] game process
Server:game92 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game91 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game89 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game87 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game88 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game90 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
[x]check [国服] game process
Server:game94 Process:myprocess Port: 8987 [dial tcp 192.168.0.195:8987: connect: connection refused]
Server:game94 Process:myprocess Port: 9101 [dial tcp 192.168.0.195:9101: connect: connection refused]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
[x]result of start [国服] game process
stop game process success

# 执行关闭后，校验时进程数量以及端口存在不符合预期时，会一直查询直到超时或者符合预期
# 直到超时都没有符合预期
[root@adver opt]# ./gamepctl apply -f stop.yaml stop
[x]stop [国服] game process
Server:game94 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game107 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game108 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:gate-db1 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game-db10 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game53 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game54 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
[x]check [国服] game process
Server:game94 Process:myprocess Port: 8987 [dial tcp 192.168.0.195:8987: connect: connection refused]
Server:game94 Process:myprocess Port: 9100 [checked]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
Server:game94 Process:myprocess Port: 8987 [dial tcp 192.168.0.195:8987: connect: connection refused]
Server:game94 Process:myprocess Port: 9100 [checked]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
Server:game94 Process:myprocess Port: 8987 [dial tcp 192.168.0.195:8987: connect: connection refused]
Server:game94 Process:myprocess Port: 9100 [checked]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
Server:game94 Process:myprocess Port: 8987 [dial tcp 192.168.0.195:8987: connect: connection refused]
Server:game94 Process:myprocess Port: 9100 [checked]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
Server:game94 Process:myprocess Port: 8987 [dial tcp 192.168.0.195:8987: connect: connection refused]
Server:game94 Process:myprocess Port: 9100 [checked]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
stopped checking process,beacuse of timeout
[x]result of stop [国服] game process
Server:game94 Process:myprocess:9100 [Online]
stop game process failed

# 执行关闭后，校验时进程数量以及端口存在不符合预期时，会一直查询直到超时或者符合预期
# 超时前都正常关闭了
[root@adver opt]# ./gamepctl apply -f stop.yaml stop
[x]stop [国服] game process
Server:game94 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game107 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:game108 Type:ssh Command:[cd /tmp && date >>time.date] [Success]
Server:gate-db1 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game-db10 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game53 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
Server:game54 Type:ssh Command:[cd /tmp && date >> time.date] [Success]
[x]check [国服] game process
Server:game94 Process:myprocess Port: 7781 [checked]
Server:game94 Process:myprocess Port: 9101 [dial tcp 192.168.0.195:9101: connect: connection refused]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
Server:game94 Process:myprocess Port: 7781 [dial tcp 192.168.0.195:7781: connect: connection refused]
Server:game94 Process:myprocess Port: 9101 [dial tcp 192.168.0.195:9101: connect: connection refused]
Server:game107 Process:hostwatchs ProcessNumber: 0 [checked]
[x]result of stop [国服] game process
stop game process success
```   
