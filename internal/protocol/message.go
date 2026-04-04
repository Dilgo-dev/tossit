package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io/fs"
	"math"
)

const (
	MsgRegister    byte = 0x01
	MsgJoin        byte = 0x02
	MsgReady       byte = 0x03
	MsgData        byte = 0x04
	MsgError       byte = 0x05
	MsgClose       byte = 0x06
	MsgBrowserJoin byte = 0x07
	MsgStored      byte = 0x08
	MsgDeleteOK    byte = 0x09
	MsgApprovalReq byte = 0x0A
	MsgApprove     byte = 0x0B
	MsgReject      byte = 0x0C

	PeerMetadata  byte = 0x10
	PeerChunk     byte = 0x11
	PeerDone      byte = 0x12
	PeerResumeReq byte = 0x13
	PeerAck       byte = 0x14

	PeerP2POffer  byte = 0x20
	PeerP2PAccept byte = 0x21
	PeerP2PReject byte = 0x22
)

type Candidate struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Type string `json:"type"`
}

func EncodeCandidates(candidates []Candidate) ([]byte, error) {
	return json.Marshal(candidates)
}

func DecodeCandidates(data []byte) ([]Candidate, error) {
	var candidates []Candidate
	if err := json.Unmarshal(data, &candidates); err != nil {
		return nil, err
	}
	return candidates, nil
}

type Message struct {
	Type    byte
	Payload []byte
}

type Metadata struct {
	Name      string      `json:"name"`
	Size      int64       `json:"size"`
	Mode      fs.FileMode `json:"mode"`
	IsDir     bool        `json:"is_dir"`
	ChunkSize int         `json:"chunk_size"`
	FileCount int         `json:"file_count,omitempty"`
}

func Encode(msg Message) []byte {
	buf := make([]byte, 1+len(msg.Payload))
	buf[0] = msg.Type
	copy(buf[1:], msg.Payload)
	return buf
}

func Decode(data []byte) (Message, error) {
	if len(data) == 0 {
		return Message{}, errors.New("empty message")
	}
	return Message{Type: data[0], Payload: data[1:]}, nil
}

func EncodeMetadata(m Metadata) ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 1+len(data))
	buf[0] = PeerMetadata
	copy(buf[1:], data)
	return buf, nil
}

func DecodeMetadata(payload []byte) (Metadata, error) {
	if len(payload) == 0 || payload[0] != PeerMetadata {
		return Metadata{}, errors.New("not a metadata message")
	}
	var m Metadata
	if err := json.Unmarshal(payload[1:], &m); err != nil {
		return Metadata{}, err
	}
	return m, nil
}

func EncodeChunk(seq uint32, ciphertext []byte) []byte {
	buf := make([]byte, 1+4+len(ciphertext))
	buf[0] = PeerChunk
	binary.BigEndian.PutUint32(buf[1:5], seq)
	copy(buf[5:], ciphertext)
	return buf
}

func DecodeChunk(payload []byte) (seq uint32, ciphertext []byte, err error) {
	if len(payload) < 5 || payload[0] != PeerChunk {
		return 0, nil, errors.New("not a chunk message")
	}
	seq = binary.BigEndian.Uint32(payload[1:5])
	ciphertext = payload[5:]
	return seq, ciphertext, nil
}

func EncodeDone(hash []byte) []byte {
	buf := make([]byte, 1+len(hash))
	buf[0] = PeerDone
	copy(buf[1:], hash)
	return buf
}

func DecodeDone(payload []byte) ([]byte, error) {
	if len(payload) < 1 || payload[0] != PeerDone {
		return nil, errors.New("not a done message")
	}
	return payload[1:], nil
}

func EncodeResumeReq(offset int64) []byte {
	if offset < 0 {
		offset = 0
	}
	buf := make([]byte, 9)
	buf[0] = PeerResumeReq
	binary.BigEndian.PutUint64(buf[1:], uint64(offset))
	return buf
}

func DecodeResumeReq(payload []byte) (int64, error) {
	if len(payload) != 9 || payload[0] != PeerResumeReq {
		return 0, errors.New("not a resume request")
	}
	v := binary.BigEndian.Uint64(payload[1:])
	if v > uint64(math.MaxInt64) {
		return 0, errors.New("offset overflow")
	}
	return int64(v), nil
}

func EncodeAck(seq uint32) []byte {
	buf := make([]byte, 5)
	buf[0] = PeerAck
	binary.BigEndian.PutUint32(buf[1:], seq)
	return buf
}

func DecodeAck(payload []byte) (uint32, error) {
	if len(payload) != 5 || payload[0] != PeerAck {
		return 0, errors.New("not an ack message")
	}
	return binary.BigEndian.Uint32(payload[1:]), nil
}
