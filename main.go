package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"net"
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

var types = map[byte]ReplyType{0x41: ReplyType{"A", "Command"},
	0x42: ReplyType{"B", "Command reply"},
	0x43: ReplyType{"C", "Get current parameter from a monitor"},
	0x44: ReplyType{"D", "Get parameter reply"},
	0x45: ReplyType{"E", "Set parameter"},
	0x46: ReplyType{"F", "Set parameter reply"}}

// possible result values in bytes 2-3 in get param reply
const successfulStatus = 0x00
const errOperationNotSupported = 0x01

type ParamReply struct {
	Header
	Message
	OPCode
	OPCodePage
	Type  [2]byte
	Value [4]byte // current value or requested value, depending on what orig msg was
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
	Message
	OPCode
	OPCodePage
	SetValue [4]byte
}

type OPCode [2]byte
type OPCodePage [2]byte

// SetValueMessage represents the message sent when setting a parameter
// TODO: figure out better abstraction
type SetValueMessage struct {
	Message
	SetValue []byte
}

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
	// TODO: add set value stuff here later...

	var result *byte
	for _, num := range checkCodeVals {
		if result == nil {
			result = new(byte)
			*result = num
			continue
		}
		*result ^= num
	}
	fmt.Println(*result)
	p.CheckCode = *result
}

func main() {
	conn, err := net.Dial("tcp", "192.168.1.101:7142")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("connected")

	pkt := ParamCommand{Delimiter: delimiter}
	// setting this for now...will need to calculate
	messageLength := new([2]byte)
	hex.Decode(messageLength[:], []byte("3036"))
	pkt.Header = Header{
		Start:         soh,
		Reserved:      reserved,
		Destination:   monitorID,
		Source:        sender,
		MessageType:   0x41,
		MessageLength: *messageLength,
	}

	opCode := new([2]byte)
	opCodePage := new([2]byte)
	hex.Decode(opCodePage[:], []byte("4332"))
	hex.Decode(opCode[:], []byte("3136"))
	// get serial number
	pkt.Message = SetParamMessage{
		OPCodePage: *opCodePage, // corresp. to 'C2'
		OPCode:     *opCode,     // '16'
	}
	pkt.Message.Start, pkt.Message.End = stx, etx

	pkt.genCheckCode()
	fmt.Println(fmt.Sprintf("%#+v", pkt))

	pkt.Send(conn)

	status, err := bufio.NewReader(conn).ReadString(delimiter)
	if err != nil {
		log.Fatal(err)
	}
	reply := ParamReply{}
	reply.MessageLength = *new([2]byte)
	reply.OPCode = OPCode(*new([2]byte))
	reply.OPCodePage = OPCodePage(*new([2]byte))
	replyHeader := status[:7]
	replyType := types[replyHeader[4]].ASCIIKey
	fmt.Println("replyType: ", replyType)
	encodedMsgLength := replyHeader[4:6]
	s, _ := hex.DecodeString(string(encodedMsgLength))
	fmt.Println("encodedMsgLength: ", s)

	ba, err := hex.DecodeString(string(status[12:32]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ba)
	for _, char := range ba {
		fmt.Print(string(char))
	}
	fmt.Println()
	fmt.Printf("%q\n", status[12:32])
}
