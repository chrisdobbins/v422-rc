package ir

var command [4]byte = [4]byte{0x43, 0x32, 0x31, 0x30} // op page code, op code
var volumeUp [4]byte = [4]byte{0x30, 0x30, 0x31, 0x37}
var volumeDown [4]byte = [4]byte{0x30, 0x30, 0x31, 0x36}

var changeInputToHDMI [4]byte = [4]byte{0x30, 0x30, 0x34, 0x32}
var changeInputToDisplayport [4]byte = [4]byte{0x30, 0x30, 0x35, 0x35}
var changeInputToDVI [4]byte = [4]byte{0x30, 0x30, 0x32, 0x44}

var togglePower [4]byte = [4]byte{0x30, 0x30, 0x30, 0x33}

var repeat []byte = []byte{0x30, 0x33} // needed to simulate IR remote

type Data struct {
	Action [4]byte
	Repeat []byte
}

func TogglePower() []byte {
	data := append([]byte{}, command[:]...)
	data = append(data, togglePower[:]...)
	data = append(data, repeat...)
	return data
}
