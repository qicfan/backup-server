# backup-server
照片备份应用的服务端，交流Q群：1055648718
- 使用websockt来上传文件，减少客户端的连接开销并且支持客户端流式传输文件
- 会使用定时任务定期扫描/upload目录，将所有照片和视频入库，客户端可以获取照片列表，然后查看、下载等
- 给客户端提供jwt验证
- 给客户端提供/upload目录的子目录列表，方便选择备份目录
- 给客户端提供创建目录接口
- 客户端访问照片列表时默认返回缩略图，缩略图会缓存下来供下次使用
- 照片或视频如果大于10MB会改为流式传输，降低服务器内存占用
- 下载时如果是华为设备导入苹果动图，会将HEIC转为JPG，MOV转为MP4
- 支持备份鸿蒙的动态照片

#### 本项目暂时没有UI，需要配合备份客户端使用：[https://github.com/qicfan/backup](https://github.com/qicfan/backup)

## 使用 Docker命令 部署

```bash
docker run -d \
	--name backup-server \
	-p 12334:12334 \
  -e TZ=Asia/Shanghai \
	-e USERNAME=admin \
	-e PASSWORD=admin \
  -e PORT=12334 \
  -e UPLOAD_ROOT_DIR=/upload \
	-v /your/upload:/upload \
	-v /your/config:/app/config \
	qicfan/backup-server:latest
```


## 目录映射说明

| 容器内路径    | 宿主机路径                           | 说明                     |
| ------------- | ------------------------------------ | ------------------------ |
| `/app/config` | `/your/config` | 配置文件、数据、日志目录   |
| `/upload`      | `/your/upload`                    | 存放照片的目录 |

## 环境变量

| 变量名 | 默认值          | 说明     |
| ------ | --------------- | -------- |
| `TZ`   | `Asia/Shanghai` | 时区设置 |
| `USERNAME`   | `admin` | 登录的用户名 |
| `PASSWORD`   | `admin` | 登录的密码 |
| `PORT`   | `12334` | WEB服务的端口号，不要改动除非有特殊需求 |
| `UPLOAD_ROOT_DIR`   | `/upload` | 上传文件的根目录，不要改动除非有特殊需求 |

## 端口说明

- **12334**: Web 服务端口

## 版本标签

- `latest` - 最新发布版本
- `v1.0.0` - 具体版本号（对应 GitHub Release）

## 首次使用

启动容器后访问，然后使用客户端APP链接yourip:12334，输入用户名，密码

## 数据备份

日志、缩略图、转码的视频都位于 `/your/config` 目录，请定期备份


## 使用 Docker Compose 部署

`docker-compose.yml`，内容如下：

```yaml
services:
  backup-server:
    image: qicfan/backup-server:latest
    container_name: backup-server
    ports:
      - "12334:12334"
    environment:
      - TZ=Asia/Shanghai
      - USERNAME=admin
      - PASSWORD=admin
      - PORT=12334
      - UPLOAD_ROOT_DIR=/upload
    volumes:
      - /your/upload:/upload
      - /your/config:/app/config
    restart: unless-stopped
```

如需启用 SSL，请将证书文件（server.crt 和 server.key）放入 `/your/config`，然后用证书对应的域名访问即可。
