# 图片存储说明

## 目录结构

图片保存在 `UPLOAD_DIR`（默认 `./uploads`），**仅两级**：业务类型 + 文件名（不再按日记 id 建子文件夹）。

```
uploads/
  travel-diaries/
    1-1.jpg
    1-2.jpg
    4-9.jpg
  daily-diaries/
    7-1.jpg
    7-2.jpg
```

文件名规则：`{日记id}-{序号}.扩展名`

对外访问路径：

- `/uploads/travel-diaries/1-1.jpg`
- `/uploads/daily-diaries/7-2.jpg`

## 从旧嵌套目录迁移

若之前已是 `uploads/travel-diaries/1/1.jpg` 这种结构，执行：

```bash
cd love-diary-go
go run ./cmd/flatten-uploads -dry-run
go run ./cmd/flatten-uploads
```

## 从 base64 迁移

```bash
go run ./cmd/migrate-images -dry-run
go run ./cmd/migrate-images
```

## 新上传

创建/编辑日记时前端传 base64，后端自动落盘到对应二级目录。

可选接口：

- `POST /upload/image?category=travel-diaries&diary_id=1`
- `POST /upload/images?category=daily-diaries&diary_id=5`

## 环境变量

| 变量 | 说明 |
|------|------|
| `UPLOAD_DIR` | 磁盘目录，默认 `./uploads` |
