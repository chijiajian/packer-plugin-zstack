package utils

func MBToBytes(mb int64) int64 {
	return mb * 1024 * 1024
}

func BytesToMB(bytes int64) int64 {
	return bytes / (1024 * 1024)
}

func BytesToGB(bytes int64) int64 {
	return bytes / (1024 * 1024 * 1024)
}

func GBToBytes(gb int64) int64 {
	return gb * 1024 * 1024 * 1024
}
