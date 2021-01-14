package protocol

import (
	"github.com/plhwin/gosocket/util"
)

// Compressor defines a common compression interface.
type Compressor interface {
	Zip([]byte) ([]byte, error)
	Unzip([]byte) ([]byte, error)
}

// RawDataCompressor implements gzip compressor for None compress
type RawDataCompressor struct {
}

func (c RawDataCompressor) Zip(data []byte) ([]byte, error) {
	return data, nil
}

func (c RawDataCompressor) Unzip(data []byte) ([]byte, error) {
	return data, nil
}

// GzipCompressor implements gzip compressor.
type GzipCompressor struct {
}

func (c GzipCompressor) Zip(data []byte) ([]byte, error) {
	return util.Zip(data)
}

func (c GzipCompressor) Unzip(data []byte) ([]byte, error) {
	return util.Unzip(data)
}

// FLateCompressor implements gzip compressor.
type FLateCompressor struct {
}

func (c FLateCompressor) Zip(data []byte) ([]byte, error) {
	return util.ZipFLate(data)
}

func (c FLateCompressor) Unzip(data []byte) ([]byte, error) {
	return util.UnzipFLate(data)
}

// SnappyCompressor implements snappy compressor
type SnappyCompressor struct {
}

func (c *SnappyCompressor) Zip(data []byte) ([]byte, error) {
	return util.ZipSnappy(data)
}

func (c *SnappyCompressor) Unzip(data []byte) ([]byte, error) {
	return util.UnzipSnappy(data)
}
