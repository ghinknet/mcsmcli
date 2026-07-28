package mcsm

// ---- Dashboard / Overview ----

// Overview corresponds to the data from GET /api/overview.
type Overview struct {
	Version                string     `json:"version"`
	SpecifiedDaemonVersion string     `json:"specifiedDaemonVersion"`
	Process                RawMessage `json:"process"`
	Record                 struct {
		Logined       int `json:"logined"`
		IllegalAccess int `json:"illegalAccess"`
		BanIPs        int `json:"banips"`
		LoginFailed   int `json:"loginFailed"`
	} `json:"record"`
	System struct {
		Type     string  `json:"type"`
		Version  string  `json:"version"`
		Node     string  `json:"node"`
		Hostname string  `json:"hostname"`
		Platform string  `json:"platform"`
		Release  string  `json:"release"`
		Uptime   float64 `json:"uptime"`
		CPU      float64 `json:"cpu"`
		TotalMem int64   `json:"totalmem"`
		FreeMem  int64   `json:"freemem"`
		Time     int64   `json:"time"`
	} `json:"system"`
	RemoteCount struct {
		Available int `json:"available"`
		Total     int `json:"total"`
	} `json:"remoteCount"`
	Remote []Daemon `json:"remote"`
}

// Daemon corresponds to node (daemon) information.
type Daemon struct {
	UUID      string `json:"uuid"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Prefix    string `json:"prefix"`
	Available bool   `json:"available"`
	Remarks   string `json:"remarks"`
	Version   string `json:"version"`
	Instance  struct {
		Running int `json:"running"`
		Total   int `json:"total"`
	} `json:"instance"`
	System struct {
		Type     string  `json:"type"`
		Hostname string  `json:"hostname"`
		Platform string  `json:"platform"`
		CPUUsage float64 `json:"cpuUsage"`
		MemUsage float64 `json:"memUsage"`
		TotalMem int64   `json:"totalmem"`
		FreeMem  int64   `json:"freemem"`
	} `json:"system"`
}

// ---- Instance ----

// Instance status codes
const (
	StatusBusy     = -1
	StatusStopped  = 0
	StatusStopping = 1
	StatusStarting = 2
	StatusRunning  = 3
)

// StatusText converts an instance status code to a human-readable string.
func StatusText(s int) string {
	switch s {
	case StatusBusy:
		return "busy"
	case StatusStopped:
		return "stopped"
	case StatusStopping:
		return "stopping"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	default:
		return "unknown"
	}
}

// InstanceConfig corresponds to the documented Type of InstanceConfig.
type InstanceConfig struct {
	Nickname          string     `json:"nickname,omitempty"`
	StartCommand      string     `json:"startCommand,omitempty"`
	StopCommand       string     `json:"stopCommand,omitempty"`
	Cwd               string     `json:"cwd,omitempty"`
	IE                string     `json:"ie,omitempty"`
	OE                string     `json:"oe,omitempty"`
	CreateDatetime    int64      `json:"createDatetime,omitempty"`
	LastDatetime      int64      `json:"lastDatetime,omitempty"`
	Type              string     `json:"type,omitempty"`
	Tag               []string   `json:"tag,omitempty"`
	EndTime           int64      `json:"endTime,omitempty"`
	FileCode          string     `json:"fileCode,omitempty"`
	ProcessType       string     `json:"processType,omitempty"`
	UpdateCommand     string     `json:"updateCommand,omitempty"`
	ActionCommandList []string   `json:"actionCommandList,omitempty"`
	CRLF              int        `json:"crlf,omitempty"`
	Docker            RawMessage `json:"docker,omitempty"`
	EnableRcon        bool       `json:"enableRcon,omitempty"`
	RconPassword      string     `json:"rconPassword,omitempty"`
	RconPort          int        `json:"rconPort,omitempty"`
	RconIP            string     `json:"rconIp,omitempty"`
	TerminalOption    RawMessage `json:"terminalOption,omitempty"`
	EventTask         RawMessage `json:"eventTask,omitempty"`
	PingConfig        RawMessage `json:"pingConfig,omitempty"`
}

// InstanceDetail corresponds to the documented Type of InstanceDetail.
type InstanceDetail struct {
	InstanceUUID string         `json:"instanceUuid"`
	Status       int            `json:"status"`
	Started      int            `json:"started"`
	Space        int64          `json:"space"`
	Config       InstanceConfig `json:"config"`
	Info         struct {
		CurrentPlayers int    `json:"currentPlayers"`
		MaxPlayers     int    `json:"maxPlayers"`
		Version        string `json:"version"`
		FileLock       int    `json:"fileLock"`
		OpenFrpStatus  bool   `json:"openFrpStatus"`
	} `json:"info"`
	ProcessInfo struct {
		CPU     float64 `json:"cpu"`
		Memory  int64   `json:"memory"`
		PID     int     `json:"pid"`
		Elapsed int64   `json:"elapsed"`
	} `json:"processInfo"`
}

// InstancePage is the paginated instance list structure.
type InstancePage struct {
	MaxPage  int              `json:"maxPage"`
	PageSize int              `json:"pageSize"`
	Data     []InstanceDetail `json:"data"`
}

// InstanceRef is an {instanceUuid, daemonId} pair used in batch operations.
type InstanceRef struct {
	InstanceUUID string `json:"instanceUuid"`
	DaemonID     string `json:"daemonId"`
}

// ---- Users ----

// User corresponds to a panel user record.
type User struct {
	UUID         string        `json:"uuid"`
	UserName     string        `json:"userName"`
	Permission   int           `json:"permission"` // 1=User, 10=Admin, -1=Banned
	RegisterTime string        `json:"registerTime"`
	LoginTime    string        `json:"loginTime"`
	Instances    []InstanceRef `json:"instances"`
	IsInit       bool          `json:"isInit"`
	Open2FA      bool          `json:"open2FA"`
	APIKey       string        `json:"apiKey"`
	Secret       string        `json:"secret"`
	PassWordType int           `json:"passWordType"`
}

// PermissionText converts a user permission code to a human-readable string.
func PermissionText(p int) string {
	switch p {
	case 10:
		return "admin"
	case 1:
		return "user"
	case -1:
		return "banned"
	default:
		return "unknown"
	}
}

// UserPage is the paginated user list structure.
type UserPage struct {
	Data     []User `json:"data"`
	MaxPage  int    `json:"maxPage"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Total    int    `json:"total"`
}

// ---- Files ----

// FileItem corresponds to a single file/directory entry.
type FileItem struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Time string `json:"time"`
	Mode int    `json:"mode"`
	Type int    `json:"type"` // 0 = directory, 1 = file
}

// FilePage is the paginated file list structure.
type FilePage struct {
	Items        []FileItem `json:"items"`
	Page         int        `json:"page"`
	PageSize     int        `json:"pageSize"`
	Total        int        `json:"total"`
	AbsolutePath string     `json:"absolutePath"`
}

// TransferCred is a one-time credential for file upload/download.
type TransferCred struct {
	Password string `json:"password"`
	Addr     string `json:"addr"`
}
