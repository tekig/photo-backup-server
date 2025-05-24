# photo-backup-server

Server for generating previews of images and videos, as well as building a database with metainformation about media files.

# Run
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
REGION=ru-central1
GATEWAY=YA-S3-TRIGGER

```
Only dev test:
```
PORT=8080
```
2. Config trigger

# Infrastructure
```
meta/
  meta.json
  d1b846df.json
origin/
  img.heic
preview/
  img.heic.jpg
```

# Infrastructure Overview

This project defines a dual-trigger infrastructure for processing media content in two stages: `origin` and `meta`.

## Triggers

There are two separate triggers:

- **Origin Trigger**
  - Can be executed in **multiple parallel threads**.
  - Responsible for processing incoming media.
  - Generates preview versions of uploaded photos.
  
- **Meta Trigger**
  - Must run in a **single thread** only.
  - Responsible for consolidating all changes, resolving them based on timestamp priority.

## Workflow

1. When media is uploaded via the `origin` trigger, a corresponding entry is created in `meta` with a **unique name**.
2. The `origin` trigger handles image processing, including preview generation.
3. The `meta` trigger collects and merges all updates into a single unified record. It prioritizes the most recent changes.

## Deletion Handling

Due to the need to distinguish between deleted media and outdated changes, the `meta` trigger supports **soft deletion only**. This allows tracking of removed items without permanently erasing the data.

---

This structure ensures scalable, safe, and time-consistent processing of media uploads and metadata synchronization.

