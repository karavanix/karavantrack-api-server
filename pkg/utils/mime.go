package utils

import "strings"

var MimeTypes = map[string]string{
	"doc":  "application/msword",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xls":  "application/vnd.ms-excel",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"ppt":  "application/vnd.ms-powerpoint",
	"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"pdf":  "application/pdf",
	"txt":  "text/plain",
	"rtf":  "application/rtf",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
	"bmp":  "image/bmp",
	"tiff": "image/tiff",
	"heic": "image/heic",
	"heif": "image/heif",
	"avif": "image/avif",
	"svg":  "image/svg+xml",
	"html": "text/html",
	"htm":  "text/html",
	"css":  "text/css",
	"js":   "application/javascript",
	"json": "application/json",
	"xml":  "application/xml",
	"zip":  "application/zip",
	"rar":  "application/x-rar-compressed",
	"tar":  "application/x-tar",
	"gz":   "application/gzip",
}

func DetectMimeByExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))

	if mime, ok := MimeTypes[ext]; ok {
		return mime
	}

	return "application/octet-stream"
}
