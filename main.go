package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
)

const delimiter = "0D"
const soh = "01"
const stx = "02"
const etx = "03"
const monitorID = "41"
const sender = "30"
const reserved = "30"

type Packet struct {
	Header    // 7 bytes
	Message   // 6-10 bytes
	CheckCode // xor every bit in message except soh
	Delimiter
}

type Header struct {
	Start         string
	Reserved      string
	MonitorID     string
	Sender        string
	MessageType   string
	MessageLength string
}

type Message struct {
	Start      string
	OpCodePage string
	OpCode     string
	End        string
}

type SetMessage struct {
	Message
	Parameter string
}

type CheckCode string
type Delimiter string

func genCheckCode(packet Packet) Packet {
	// packet.Header (except for SOH), packet.Message
	stuffToDecode := []int{0x30, 0x41, 0x30, 0x41, 0x30, 0x43, 0x02, 0x43, 0x32, 0x31, 0x30, 0x30, 0x30, 0x31, 0x37, 0x30, 0x33, 0x03}
	var result *int
	for _, num := range stuffToDecode {
		if result == nil {
			result = new(int)
			*result = num
			continue
		}
		*result ^= num
	}
	fmt.Println(*result)
}

func main() {
	conn, err := net.Dial("tcp", "192.168.1.101:7142")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("connected")
	// increases volume
	hexVals := "01 30 41 30 41 30 43 02 43 32 31 30 30 30 31 37 30 33 03 07 0D"
	pkt := Packet{
		Header{
			Start:         soh,
			Reserved:      reserved,
			MonitorID:     monitorID,
			Sender:        sender,
			MessageType:   "41",   // corresp. to ascii 'A', which indic a message of type "Command"
			MessageLength: "3036", // "30" -> '0', "36" -> '6'
		},
		Message{
			Start:      stx,
			OpCodePage: "4332", // corresp. to 'C2'
			OpCode:     "3136", // '16'
			End:        etx,
		},
		CheckCode: genCheckCode(),
		Delimiter: delimiter,
	}
	out := []byte{}
	for _, val := range strings.Split(hexVals, " ") {
		decVal, err := hex.DecodeString(val)
		if err != nil {
			log.Fatal(err)
		}
		out = append(out, decVal...)
	}
	conn.Write(out)
}
