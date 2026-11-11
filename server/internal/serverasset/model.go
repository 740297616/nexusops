package serverasset

import "time"

const (
	StatusPending  = "pending"
	StatusOnline   = "online"
	StatusOffline  = "offline"
	StatusDisabled = "disabled"
)

type Server struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Hostname        string     `json:"hostname"`
	Description     string     `json:"description"`
	OS              string     `json:"os"`
	OSVersion       string     `json:"os_version"`
	Architecture    string     `json:"architecture"`
	KernelVersion   string     `json:"kernel_version"`
	CPUModel        string     `json:"cpu_model"`
	CPUCores        int        `json:"cpu_cores"`
	MemoryTotal     int64      `json:"memory_total"`
	DiskTotal       int64      `json:"disk_total"`
	PrimaryIP       string     `json:"primary_ip"`
	AgentVersion    string     `json:"agent_version"`
	AgentStatus     string     `json:"agent_status"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
	LastReportAt    *time.Time `json:"last_report_at"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (Server) TableName() string { return "servers" }

type AgentCredentialInfo struct {
	ServerID  string     `json:"server_id"`
	Status    string     `json:"status"`
	IssuedAt  time.Time  `json:"issued_at"`
	RotatedAt *time.Time `json:"rotated_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

type AgentPrincipal struct {
	ServerID     string
	CredentialID int64
}

type Page struct {
	Items    []Server `json:"items"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

