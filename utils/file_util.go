package utils

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"github.com/banbox/banbot/btime"
	"github.com/banbox/banbot/core"
	"github.com/banbox/banexg"
	"github.com/banbox/banexg/errs"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func CopyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err = EnsureDir(dst, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		fileInfo, err := os.Lstat(sourcePath)
		if err != nil {
			return err
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err = EnsureDir(destPath, 0755); err != nil {
				return err
			}
			if err = CopyDir(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			if err = CopySymLink(sourcePath, destPath); err != nil {
				return err
			}
		default:
			if err = Copy(sourcePath, destPath); err != nil {
				return err
			}
		}

		fInfo, err := entry.Info()
		if err != nil {
			return err
		}

		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err = os.Chmod(destPath, fInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func Copy(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func Exists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func EnsureDir(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func CopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}

func WriteCsvFile(path string, rows [][]string, compress bool) *errs.Error {
	var fileWriter io.Writer
	var err_ error
	if compress {
		zipFile, err_ := os.Create(strings.Replace(path, ".csv", ".zip", 1))
		if err_ != nil {
			return errs.New(errs.CodeIOWriteFail, err_)
		}
		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()
		header := &zip.FileHeader{
			Name:     filepath.Base(path),
			Method:   zip.Deflate,
			Modified: time.Now(),
		}
		fileWriter, err_ = zipWriter.CreateHeader(header)
		if err_ != nil {
			return errs.New(errs.CodeIOWriteFail, err_)
		}
	} else {
		file, err_ := os.Create(path)
		if err_ != nil {
			return errs.New(errs.CodeIOWriteFail, err_)
		}
		defer file.Close()
		fileWriter = file
	}
	writer := csv.NewWriter(fileWriter)
	defer writer.Flush()
	err_ = writer.WriteAll(rows)
	if err_ != nil {
		return errs.New(errs.CodeIOWriteFail, err_)
	}
	return nil
}

func KlineToStr(klines []*banexg.Kline, loc *time.Location) [][]string {
	rows := make([][]string, 0, len(klines))
	for _, k := range klines {
		var dateStr string
		if loc != nil {
			dateStr = btime.ToTime(k.Time).In(loc).Format(core.DefaultDateFmt)
		} else {
			dateStr = strconv.FormatInt(k.Time/1000, 10)
		}
		row := []string{
			dateStr,
			strconv.FormatFloat(k.Open, 'f', -1, 64),
			strconv.FormatFloat(k.High, 'f', -1, 64),
			strconv.FormatFloat(k.Low, 'f', -1, 64),
			strconv.FormatFloat(k.Close, 'f', -1, 64),
			strconv.FormatFloat(k.Volume, 'f', -1, 64),
			strconv.FormatFloat(k.Info, 'f', -1, 64),
		}
		rows = append(rows, row)
	}
	return rows
}

func ReadLastNLines(filePath string, lineCount int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []string

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	// 设定缓冲区大小和读取偏移量
	bufferSize := 4096
	var offset = fileSize
	var buffer []byte

	var tmp string
	for offset > 0 && len(result) < lineCount {
		if int64(bufferSize) > offset {
			bufferSize = int(offset)
			offset = 0
		} else {
			offset -= int64(bufferSize)
		}

		buffer = make([]byte, bufferSize)
		_, err = file.ReadAt(buffer, offset)
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(buffer), "\n")
		if len(lines) > 0 {
			lines[len(lines)-1] += tmp
			tmp = lines[0]
			lines = lines[1:]
		} else {
			tmp = ""
		}
		// 倒序读取行
		for i := len(lines) - 1; i >= 0; i-- {
			if len(result) < lineCount {
				if lines[i] != "" {
					result = append(result, lines[i])
				}
			} else {
				break
			}
		}
	}

	// 倒序返回结果
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}
