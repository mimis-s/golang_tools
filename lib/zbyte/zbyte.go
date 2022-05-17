package zbyte

func BigEndByteToInt32(arr []byte) int {
	var Size int
	Size += int(arr[3])
	Size += int(arr[2] << 8)
	Size += int(arr[1] << 16)
	Size += int(arr[0] << 24)
	return Size
}

func BigEndInt32ToByte(data int) []byte {
	PkgIDByte := make([]byte, 4)
	PkgIDByte[3] = uint8(data)
	PkgIDByte[2] = uint8(data >> 8)
	PkgIDByte[1] = uint8(data >> 16)
	PkgIDByte[0] = uint8(data >> 24)
	return PkgIDByte
}

func SmallEndByteToInt32(arr []byte) int {
	var Size int
	Size += int(arr[0])
	Size += int(arr[1] << 8)
	Size += int(arr[2] << 16)
	Size += int(arr[3] << 24)
	return Size
}

func SmallEndInt32ToByte(data int) []byte {
	PkgIDByte := make([]byte, 4)
	PkgIDByte[0] = uint8(data)
	PkgIDByte[1] = uint8(data >> 8)
	PkgIDByte[2] = uint8(data >> 16)
	PkgIDByte[3] = uint8(data >> 24)
	return PkgIDByte
}

func SmallEndByteToInt64(arr []byte) int64 {
	var Size int64
	Size += int64(arr[0])
	Size += int64(arr[1] << 8)
	Size += int64(arr[2] << 16)
	Size += int64(arr[3] << 24)
	Size += int64(arr[4] << 32)
	Size += int64(arr[5] << 40)
	Size += int64(arr[6] << 48)
	Size += int64(arr[7] << 56)
	return Size
}

func SmallEndInt64ToByte(data int64) []byte {
	PkgIDByte := make([]byte, 8)
	PkgIDByte[0] = uint8(data)
	PkgIDByte[1] = uint8(data >> 8)
	PkgIDByte[2] = uint8(data >> 16)
	PkgIDByte[3] = uint8(data >> 24)
	PkgIDByte[4] = uint8(data >> 32)
	PkgIDByte[5] = uint8(data >> 40)
	PkgIDByte[6] = uint8(data >> 48)
	PkgIDByte[7] = uint8(data >> 56)
	return PkgIDByte
}
