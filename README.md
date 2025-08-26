# backup-server
照片备份应用的服务端
- 使用websockt来上传文件，减少客户端的连接开销并且支持客户端流式传输文件
- 会使用定时任务定期扫描/upload目录，将所有照片和视频入库，客户端可以获取照片列表，然后查看、下载、删除等
- 给客户端提供jwt验证
- 给客户端提供/upload目录的子目录列表，方便选择备份目录
- 给客户端提供创建目录接口
- 可以识别华为和苹果的动态照片
- 鸿蒙客户端可以下载苹果和华为的动态照片并导入相册
- 客户端访问照片列表时默认返回缩略图，缩略图会缓存下来供下次使用
- 视频不会显示缩略图，由客户端负责渲染一个首帧画面展示在列表中


## 使用 Docker命令 构建容器

```bash
docker run -d \
	--name backup-server \
	-p 12334:12334 \
	-e USERNAME=admin \
	-e PASSWORD=admin \
    -e PORT=12334 \
	-v /your/upload/dir:/upload \
	-v /your/config/dir:/app/config \
	qicfan/backup-server:latest
```

参数说明：
- `-p 12334:12334` 映射端口
- `-e USERNAME=admin` 设置登录用户名
- `-e PASSWORD=admin` 设置登录密码
- `-e PORT=12334` 设置访问的端口号，一般不动除非有特殊需求
- `-v /your/upload/dir:/upload` 挂载上传文件目录（也就是存放照片的目录）
- `-v /your/config/dir:/app/config` 挂载配置和日志目录

## 使用 Docker Compose 部署

`docker-compose.yml`，内容如下：

```yaml
version: '3.8'
services:
	backup-server:
		image: qicfan/backup-server:latest
		container_name: backup-server
		ports:
			- "12334:12334"
		environment:
			- USERNAME=admin
			- PASSWORD=admin
			- PORT=12334
		volumes:
			- /your/upload/dir:/upload
			- /your/config/dir:/app/config
		restart: unless-stopped
```

如需启用 SSL，请将证书文件（server.crt 和 server.key）放入 `/app/config`，然后用证书对应的域名访问即可。
