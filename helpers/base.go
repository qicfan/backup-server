package helpers

var RootDir string = ""

var UPLOAD_ROOT_DIR = "/upload"

type ClientOS string

const (
	UNKNOW  ClientOS = ""
	HMOS    ClientOS = "HMOS"
	ANDROID ClientOS = "ANDROID"
	IOS     ClientOS = "IOS"
)
