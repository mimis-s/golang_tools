package register

var MapIDFunc = make(map[int]func([]byte))

func RegisterEvent(eventID int, callBack func([]byte)) {
	MapIDFunc[eventID] = callBack
}

func CallEvent(eventID int, args []byte) {
	MapIDFunc[eventID](args)
}
