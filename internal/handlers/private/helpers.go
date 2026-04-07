package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/uuid"

	"github.com/shurco/mycart/pkg/fsutil"
)

const (
	dirUploads  = "./lc_uploads"
	dirDigitals = "./lc_digitals"
)

var validImageMIMETypes = []string{"image/png", "image/jpeg"}

// saveFile atomically saves the uploaded file to a temporary file, then renames it.
func saveFile(file *multipart.FileHeader, filePath string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	tmpPath := filePath + ".tmp"
	if err := os.MkdirAll(filepath.Dir(filePath), 0o775); err != nil {
		return err
	}

	dst, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return err
	}

	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, filePath)
}

func validateImageMIME(mimeType string) bool {
	return slices.Contains(validImageMIMETypes, mimeType)
}

// generateFileName generates a unique file name with extension.
func generateFileName(originalName string) (fileUUID, fileExt, fileName string) {
	fileUUID = uuid.New().String()
	fileExt = fsutil.ExtName(originalName)
	fileName = fmt.Sprintf("%s.%s", fileUUID, fileExt)
	return fileUUID, fileExt, fileName
}
