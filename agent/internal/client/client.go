package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/740297616/nexusops/agent/internal/collector"
	"github.com/740297616/nexusops/agent/internal/config"
)
type Client struct { cfg config.Config; credential string; http *http.Client; serverID string }
func New(cfg config.Config)*Client{return &Client{cfg:cfg,http:&http.Client{Timeout:10*time.Second}}}
func(c *Client)SetCredential(v string){c.credential=v}
func(c *Client)endpoint(path string)string{return strings.TrimRight(c.cfg.ServerURL,"/")+path}
func(c *Client)do(ctx context.Context,path string,body any,out any,auth bool)error{raw,err:=json.Marshal(body);if err!=nil{return err};req,err:=http.NewRequestWithContext(ctx,http.MethodPost,c.endpoint(path),bytes.NewReader(raw));if err!=nil{return err};req.Header.Set("Content-Type","application/json");if auth{req.Header.Set("Authorization","Agent "+c.credential)};resp,err:=c.http.Do(req);if err!=nil{return err};defer resp.Body.Close();if resp.StatusCode/100!=2{return fmt.Errorf("server returned status %d",resp.StatusCode)};if out!=nil{return json.NewDecoder(resp.Body).Decode(out)};return nil}
func(c *Client)Register(ctx context.Context,i collector.Info)(string,error){var out struct{Data struct{ServerID string `json:"server_id"`; Credential string `json:"credential"`} `json:"data"`};err:=c.do(ctx,"/agent/register",map[string]any{"protocol_version":"1","registration_token":c.cfg.RegistrationToken,"hostname":i.Hostname,"os":i.OS,"os_version":i.OSVersion,"architecture":i.Architecture,"kernel_version":i.KernelVersion,"cpu_model":i.CPUModel,"cpu_cores":i.CPUCores,"memory_total":i.MemoryTotal,"disk_total":i.DiskTotal,"primary_ip":i.PrimaryIP,"agent_version":i.AgentVersion},&out,false);if err!=nil{return "",err};if out.Data.Credential==""{return "",errors.New("server returned empty credential")};c.serverID=out.Data.ServerID;return out.Data.Credential,nil}
func(c *Client)Heartbeat(ctx context.Context,i collector.Info)error{if c.credential==""{return errors.New("missing credential")};var last error;for attempt:=0;attempt<3;attempt++{var out any;last=c.do(ctx,"/agent/heartbeat",map[string]any{"protocol_version":"1","request_id":fmt.Sprintf("%d",time.Now().UnixNano()),"agent_version":i.AgentVersion,"timestamp":time.Now().UTC()},&out,true);if last==nil{return nil};select{case <-time.After(time.Duration(1<<attempt)*250*time.Millisecond):case <-ctx.Done():return ctx.Err()}};return last}



func (c *Client) Report(ctx context.Context, i collector.Info) error {
	if c.credential == "" { return errors.New("missing credential") }
	var out any
	return c.do(ctx, "/agent/report", map[string]any{
		"protocol_version": "1", "request_id": fmt.Sprintf("%d", time.Now().UnixNano()),
		"timestamp": time.Now().UTC(), "agent_version": i.AgentVersion, "hostname": i.Hostname,
		"os": i.OS, "os_version": i.OSVersion, "architecture": i.Architecture,
		"kernel_version": i.KernelVersion, "cpu_model": i.CPUModel, "cpu_cores": i.CPUCores,
		"memory_total": i.MemoryTotal, "disk_total": i.DiskTotal, "primary_ip": i.PrimaryIP,
	}, &out, true)
}
