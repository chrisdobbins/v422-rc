package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/chrisdobbins/v422rc/ir"
)

const delimiter = 0x0D
const soh = 0x01
const stx = 0x02
const etx = 0x03
const monitorID = 0x41
const sender = 0x30
const reserved = 0x30

var nullMessage struct {
	message     Message
	commandCode [2]byte
}

type ReplyType struct {
	ASCIIKey string
	Value    string
}

var types = map[string]byte{"A": 0x41, // "Command"
	"B": 0x42, // "Command reply"
	"C": 0x43, // "Get current parameter from a monitor"
	"D": 0x44, //"Get parameter reply"
	"E": 0x45, // "Set parameter"
	"F": 0x46} // "Set parameter reply

// possible result values in bytes 2-3 in get param reply
const successfulStatus = 0x00
const errOperationNotSupported = 0x01

type ParamReply struct {
	Header
	Message
	OPCode
	OPCodePage
	Type  [2]byte
	Value [6]byte // current value or requested value, depending on what orig msg was
}

type SaveMessage struct {
	Message
	CommandCode [2]byte
}

// ParamCommand represents the entire packet sent to the monitor when getting or setting a param
type ParamCommand struct {
	Header
	Message   SetParamMessage
	CheckCode byte // xor every bit in message except soh
	Delimiter byte
}

// Header represents the first 7 bytes of the packet sent to the monitor
type Header struct {
	Start         byte
	Reserved      byte
	Destination   byte
	Source        byte
	MessageType   byte
	MessageLength [2]byte
}

// Message represents the data being sent to the monitor
type Message struct {
	Start byte
	End   byte
}

type SetParamMessage struct {
	Start byte
	End   byte
	OPCode
	OPCodePage
	SetValue [6]byte
}

type OPCode [2]byte
type OPCodePage [2]byte

// Send writes the completed packet to the provided connection
// The packet must have this order in the byte array written to the connection:
// Header (7 bytes), Message (6-18[?] bytes), Check Code (1 byte), Delimiter (1 byte)
func (p ParamCommand) Send(conn net.Conn) {
	// order matters
	out := []byte{p.Header.Start, p.Header.Reserved, p.Header.Destination, p.Header.Source, p.Header.MessageType}
	out = append(out, p.Header.MessageLength[:]...)
	out = append(out, p.Message.Start)
	out = append(out, p.Message.OPCodePage[:]...)
	out = append(out, p.Message.OPCode[:]...)
	out = append(out, p.Message.SetValue[:]...)
	out = append(out, p.Message.End, p.CheckCode, p.Delimiter)
	conn.Write(out)
}

func (p *ParamCommand) genCheckCode() {
	// packet.Header (except for SOH), packet.Message
	// XOR operation. order in slice doesn't matter
	checkCodeVals := []byte{p.Message.Start, p.Message.End, p.Header.Destination, p.Header.MessageType, p.Header.Reserved, p.Header.Source}
	checkCodeVals = append(checkCodeVals, p.Header.MessageLength[:]...)
	checkCodeVals = append(checkCodeVals, p.Message.OPCode[:]...)
	checkCodeVals = append(checkCodeVals, p.Message.OPCodePage[:]...)
	checkCodeVals = append(checkCodeVals, p.Message.SetValue[:]...)

	var result *byte
	for _, num := range checkCodeVals {
		if result == nil {
			result = new(byte)
			*result = num
			continue
		}
		*result ^= num
	}
	p.CheckCode = *result
}

var commands = map[string]func() []byte{"toggle power": ir.TogglePower,
	"to Displayport": ir.ToDisplayport}

func (p *ParamCommand) Set(_ string) {
	cmd := commands["to Displayport"]() // commands["toggle power"]()
	p.Header.MessageType = types["A"]
	copy(p.Message.OPCodePage[:], cmd[:2])
	copy(p.Message.OPCode[:], cmd[2:4])
	copy(p.Message.SetValue[:], cmd[4:])
	formattedMessageLength := fmt.Sprintf("%02X", 2+len(cmd))
	fmt.Printf("%+v\n", p)
	for idx, ch := range formattedMessageLength {
		p.Header.MessageLength[idx] = byte(ch)
	}
}

func main() {
	conn, err := net.Dial("tcp", "192.168.1.101:7142")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("connected")

	pkt := ParamCommand{Delimiter: delimiter}

	pkt.Header.Start = soh
	pkt.Header.Reserved = reserved
	pkt.Message.Start, pkt.Message.End = stx, etx
	pkt.Destination = monitorID
	pkt.Source = sender

	pkt.Set("tempVal")
	pkt.genCheckCode()
	fmt.Println(fmt.Sprintf("%#+v", pkt))
	pkt.Send(conn)

	status, err := bufio.NewReader(conn).ReadString(delimiter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(status)
	// reply := ParamReply{}
	// reply.MessageLength = *new([2]byte)
	// reply.OPCode = OPCode(*new([2]byte))
	// reply.OPCodePage = OPCodePage(*new([2]byte))
	// replyHeader := status[:7]
	// replyType := types[replyHeader[4]].ASCIIKey
	// fmt.Println("replyType: ", replyType)
	// encodedMsgLength := replyHeader[4:6]
	// s, _ := hex.DecodeString(string(encodedMsgLength))
	// fmt.Println("encodedMsgLength: ", s)

	// ba, err := hex.DecodeString(string(status[12:32]))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(ba)
	// for _, char := range ba {
	// 	fmt.Print(string(char))
	// }
	// fmt.Println()
}
