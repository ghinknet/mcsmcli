# mcsmcli

MCSManager（MCSM）面板的 Go 命令行管理工具，基于官方 API 文档实现，覆盖文档中的全部接口：鉴权保存、面板总览、节点管理、实例全生命周期（启停重启/创建删除/命令/日志/重装）、用户管理、文件管理（含上传下载）、Docker 镜像管理。

## 构建

```bash
go build -o mcsmcli .
```

依赖 `go.gh.ink/json`、`go.gh.ink/timex`、`go.gh.ink/toolbox`（GoVanityPath，自动拉取）。

## 鉴权

API Key 在面板「用户中心」生成（权限与账号一致，管理员 Key 请妥善保管）。

```bash
# 保存鉴权（默认会调用 /api/overview 验证连通性，--no-verify 可跳过）
mcsmcli login --url https://panel.example.com --apikey <API_KEY> [--daemon <daemonId>]

mcsmcli whoami          # 查看当前配置并测试连通性
mcsmcli logout          # 删除保存的鉴权
```

配置保存在 `~/.config/mcsmcli/config.json`（权限 0600，可用 `MCSM_CONFIG` 改路径）。

多面板支持：

```bash
mcsmcli --profile prod login --url ... --apikey ...
mcsmcli profile list
mcsmcli profile use prod
mcsmcli profile set-daemon <daemonId>   # 设置默认节点，之后可省略 -d
```

环境变量 `MCSM_URL` / `MCSM_APIKEY` / `MCSM_DAEMON` 可临时覆盖配置；全局标志 `--url` / `--apikey` / `-d` 优先级最高。

## 常用命令

```bash
# 总览与节点
mcsmcli overview
mcsmcli daemon list
mcsmcli daemon add --ip 10.0.0.16 --port 24444 --key <daemonKey> --remarks 备注
mcsmcli daemon link <daemonId>
mcsmcli daemon update <daemonId> --remarks 新备注 --available true
mcsmcli daemon delete <daemonId>

# 实例（-d 可省略，若已设默认节点）
mcsmcli instance list -d <daemonId> [--name 关键词]
mcsmcli instance info <uuid>
mcsmcli instance start|stop|restart|kill <uuid>
mcsmcli instance batch-start uuid1 uuid2@其他daemonId
mcsmcli instance cmd <uuid> say hello
mcsmcli instance log <uuid> --size 64
mcsmcli instance create --file config.json      # InstanceConfig JSON，- 表示 stdin
mcsmcli instance update <uuid> --nickname 新名字
mcsmcli instance delete <uuid> [--files]        # --files 同时删除实例文件
mcsmcli instance upgrade <uuid>                 # 触发 update 命令
mcsmcli instance reinstall <uuid> --target-url https://.../server.zip

# 用户
mcsmcli user list [--role 10]
mcsmcli user create --name bob --password '...' --permission 1
mcsmcli user update <uuid> --permission 10      # 或 --file config.json 提交完整 config
mcsmcli user delete <uuid>...

# 文件（第一个参数是实例 uuid）
mcsmcli file ls <uuid> /
mcsmcli file cat <uuid> /server.properties
mcsmcli file write <uuid> /eula.txt --text 'eula=true'
mcsmcli file download <uuid> /backup/world.zip ./world.zip
mcsmcli file upload <uuid> ./plugin.jar /plugins
mcsmcli file cp|mv <uuid> <源> <目标>
mcsmcli file zip <uuid> /backup.zip /world /config
mcsmcli file unzip <uuid> /backup.zip /restore --code utf-8
mcsmcli file rm|touch|mkdir <uuid> <路径>

# Docker
mcsmcli image list|containers|networks
mcsmcli image build --name mcsm-custom --tag latest --dockerfile ./Dockerfile
mcsmcli image progress
```

所有查询命令加 `--json` 可输出面板原始 data，便于脚本处理。

## 说明

- 文件上传/下载按文档走两段式：先向面板取一次性凭据，再与 daemon 直连传输；直连协议默认跟随面板（https 面板 → https daemon），可用 `MCSM_DAEMON_SCHEME=http|https` 强制指定。
- 大文件传输不受 `--timeout`（默认 30s，仅限普通 API 请求）限制。
- `daemon update` 会先拉取节点现有配置再合并你指定的字段，未指定的保持原值（daemon 的 apiKey 面板不回传，如需保留请显式传 `--key`）。
