package qpdf

import "errors"

/**
 * Errors defines all error in opdf
 * */

var (
	// ErrLackOfInit means lack of pdf file initial
	ErrLackOfInit = errors.New("please initial a pdf, by Initial")

	// ErrLoadImageByURL means read image by url has error
	ErrLoadImageByURL = errors.New("load image by given url failed")

	// ErrImageRead means image decode process has error
	ErrImageRead = errors.New("image read from url error")

	// ErrImageWrite means ioutil write image has error
	ErrImageWrite = errors.New("ioutil write image has error")

	// ErrExportByte means export pdf []byte failure
	ErrExportByte = errors.New("export pdf in byte array failed")
)
