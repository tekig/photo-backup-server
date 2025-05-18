# photo-backup-server

Server for generating previews of images and videos, as well as building a database with metainformation about media files.

# run
## Yandex S3 Trigger
0. Build in yandex register
```
docker pull ghcr.io/tekig/photo-backup-server:<version>
docker tag ghcr.io/tekig/photo-backup-server:<version> cr.yandex/<folder>/photo-backup-server:<version>
docker push cr.yandex/<folder>/photo-backup-server:<version>
```
1. Field env from Serverless Containers
```
ENDPOINT=https://storage.yandexcloud.net
ACCESS_KEY=access key
ACCESS_SECRET=access secret
REGION=ru-central1"
GATEWAY=YA-S3-TRIGGER

```
Only dev test:
```
PORT=8080
```
2. Config trigger

