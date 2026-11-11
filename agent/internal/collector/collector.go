package collector

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type Info struct { Hostname, OS, OSVersion, Architecture, KernelVersion, CPUModel, PrimaryIP string; CPUCores int; MemoryTotal, DiskTotal int64; AgentVersion string }
func Collect() Info { i:=Info{Architecture:runtime.GOARCH,CPUCores:runtime.NumCPU()};i.Hostname,_=os.Hostname();i.OS=runtime.GOOS;i.OSVersion=readOS();i.KernelVersion=command("uname","-r");i.CPUModel=readCPU();i.MemoryTotal=readMem();i.DiskTotal=readDisk();i.PrimaryIP=readIP();return i }
func readOS()string{b,err:=os.ReadFile("/etc/os-release");if err!=nil{return runtime.GOOS};for _,line:=range strings.Split(string(b),"\n"){if strings.HasPrefix(line,"PRETTY_NAME="){return strings.Trim(strings.TrimPrefix(line,"PRETTY_NAME="),"\"")}};return runtime.GOOS}
func command(name string,args ...string)string{b,err:=exec.Command(name,args...).Output();if err!=nil{return ""};return strings.TrimSpace(string(b))}
func readCPU()string{f,err:=os.Open("/proc/cpuinfo");if err!=nil{return ""};defer f.Close();s:=bufio.NewScanner(f);for s.Scan(){if strings.HasPrefix(s.Text(),"model name"){p:=strings.SplitN(s.Text(),":",2);if len(p)==2{return strings.TrimSpace(p[1])}}};return ""}
func readMem()int64{f,err:=os.Open("/proc/meminfo");if err!=nil{return 0};defer f.Close();s:=bufio.NewScanner(f);for s.Scan(){p:=strings.Fields(s.Text());if len(p)>=2&&p[0]=="MemTotal:"{v,_:=strconv.ParseInt(p[1],10,64);return v*1024}};return 0}
func readDisk()int64{out:=command("df","-B1","/");lines:=strings.Split(out,"\n");if len(lines)>1{p:=strings.Fields(lines[len(lines)-1]);if len(p)>=2{v,_:=strconv.ParseInt(p[1],10,64);return v}};return 0}
func readIP()string{out:=command("hostname","-I");if p:=strings.Fields(out);len(p)>0{return p[0]};return ""}
