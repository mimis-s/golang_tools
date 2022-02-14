package zbyte

func ByteToInt(arr []byte) int {
	var Size int
	Size += int(arr[3])
	Size += int(arr[2] << 8)
	Size += int(arr[1] << 16)
	Size += int(arr[0] << 24)
	return Size
}

func IntToByte(data int) []byte {
	PkgIDByte := make([]byte, 4)
	PkgIDByte[3] = uint8(data)
	PkgIDByte[2] = uint8(data >> 8)
	PkgIDByte[1] = uint8(data >> 16)
	PkgIDByte[0] = uint8(data >> 24)
	return PkgIDByte
}
