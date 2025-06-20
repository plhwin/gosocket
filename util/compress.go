package util

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"github.com/golang/snappy"
	"io"
)

// zip data by gzip
func Zip(buf []byte) (data []byte, err error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	if _, err = w.Write(buf); err != nil {
		return
	}
	if err = w.Flush(); err != nil {
		return
	}
	// 不可以 defer w.Close()，否则Unzip的时候，ioutil.ReadAll 报 unexpected EOF，
	// 也就是defer没有达到写完数据后关闭的目的
	// https://stackoverflow.com/questions/34284001/having-trouble-with-gzip-reader-in-golang
	w.Close()

	return b.Bytes(), nil
}

// unzip data by gzip
func Unzip(buf []byte) (data []byte, err error) {
	var r *gzip.Reader
	c := bytes.NewBuffer(buf)
	if r, err = gzip.NewReader(c); err != nil {
		fmt.Println("gzip.NewReader error:", err)
		return
	}
	defer r.Close()
	return io.ReadAll(r)
}

// zip data by FLate
func ZipFLate(buf []byte) (data []byte, err error) {
	var b bytes.Buffer
	var w *flate.Writer
	w, err = flate.NewWriter(&b, flate.BestCompression)

	if _, err = w.Write(buf); err != nil {
		return
	}
	if err = w.Flush(); err != nil {
		return
	}
	w.Close()

	return b.Bytes(), nil
}

// unzip data by FLate
func UnzipFLate(buf []byte) (data []byte, err error) {
	r := flate.NewReader(bytes.NewBuffer(buf))
	defer r.Close()
	return io.ReadAll(r)
}

func ZipSnappy(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	var buffer bytes.Buffer
	writer := snappy.NewBufferedWriter(&buffer)
	_, err := writer.Write(data)
	if err != nil {
		writer.Close()
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func UnzipSnappy(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	reader := snappy.NewReader(bytes.NewReader(data))
	out, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return out, err
}
