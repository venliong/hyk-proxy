package event

import (
	"bytes"
	"encoding/binary"
	"codec"
	"snappy"
	"se1"
	vector "container/vector"
)

const (
	HTTP_REQUEST_EVENT_TYPE                = 1000
	HTTP_RESPONSE_EVENT_TYPE               = 1001
	HTTP_CHUNK_EVENT_TYPE                  = 1002
	HTTP_ERROR_EVENT_TYPE                  = 1003
	HTTP_CONNECTION_EVENT_TYPE             = 1004
	RESERVED_SEGMENT_EVENT_TYPE            = 48100
	COMPRESS_EVENT_TYPE                    = 1500
	ENCRYPT_EVENT_TYPE                     = 1501
	AUTH_REQUEST_EVENT_TYPE                = 2000
	AUTH_RESPONSE_EVENT_TYPE               = 2001
	USER_OPERATION_EVENT_TYPE              = 2010
	GROUP_OPERATION_EVENT_TYPE             = 2011
	USER_LIST_REQUEST_EVENT_TYPE           = 2012
	GROUOP_LIST_REQUEST_EVENT_TYPE         = 2013
	USER_LIST_RESPONSE_EVENT_TYPE          = 2014
	GROUOP_LIST_RESPONSE_EVENT_TYPE        = 2015
	BLACKLIST_OPERATION_EVENT_TYPE         = 2016
	REQUEST_SHARED_APPID_EVENT_TYPE        = 2017
	REQUEST_SHARED_APPID_RESULT_EVENT_TYPE = 2018
	ADMIN_RESPONSE_EVENT_TYPE              = 2020
	SERVER_CONFIG_EVENT_TYPE               = 2050
)

const (
	MAGIC_NUMBER uint16 = 0xCAFE
)

type EventHeaderTags struct {
	magic uint16
	token string
}

func (tags *EventHeaderTags) Encode(buffer *bytes.Buffer) bool {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, MAGIC_NUMBER)
	buffer.Write(b)
	codec.WriteVarString(buffer, tags.token)
	return true
}
func (tags *EventHeaderTags) Decode(buffer *bytes.Buffer) bool {
	b := make([]byte, 2)
	realLen, err := buffer.Read(b)
	if err != nil || realLen != 2 {
		return false
	}
	tags.magic = binary.BigEndian.Uint16(b)
	if tags.magic != MAGIC_NUMBER {
		return false
	}
	token, ok := codec.ReadVarString(buffer)
	tags.token = token
	return ok
}

type Event interface {
	Encode(buffer *bytes.Buffer) bool
	Decode(buffer *bytes.Buffer) bool
	GetType() uint32
	GetVersion() uint32
	GetHash() uint32
	SetHash(hash uint32)
	GetAttachement() interface{}
	SetAttachement(interface{})
}

type Attachement struct {
	attachment interface{}
}

func (att *Attachement) GetAttachement() interface{} {
	return att.attachment
}
func (att *Attachement) SetAttachement(a interface{}) {
	att.attachment = a
}

type HashField struct {
	Hash uint32
}

func (field *HashField) GetHash() uint32 {
	return field.Hash
}
func (field *HashField) SetHash(hash uint32) {
	field.Hash = hash
}

type EventHeader struct {
	Type    uint32
	Version uint32
	Hash    uint32
}

func (header *EventHeader) Encode(buffer *bytes.Buffer) bool {
	codec.WriteUvarint(buffer, uint64(header.Type))
	codec.WriteUvarint(buffer, uint64(header.Version))
	codec.WriteUvarint(buffer, uint64(header.Hash))
	return true
}
func (header *EventHeader) Decode(buffer *bytes.Buffer) bool {
	tmp1, err1 := codec.ReadUvarint(buffer)
	tmp2, err2 := codec.ReadUvarint(buffer)
	tmp3, err3 := codec.ReadUvarint(buffer)
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	header.Type, header.Version, header.Hash = uint32(tmp1), uint32(tmp2), uint32(tmp3)
	return true
}

func EncodeEvent(buffer *bytes.Buffer, ev Event) bool {
	header := EventHeader{ev.GetType(), ev.GetVersion(), ev.GetHash()}
	header.Encode(buffer)
	return ev.Encode(buffer)
}

func DecodeEvent(buffer *bytes.Buffer) (bool, Event) {
	return ParseEvent(buffer)
}

type AuthRequestEvent struct {
	Appid  string
	User   string
	Passwd string
	HashField
	Attachement
}

func (req *AuthRequestEvent) Encode(buffer *bytes.Buffer) bool {
	codec.WriteVarString(buffer, req.Appid)
	codec.WriteVarString(buffer, req.User)
	codec.WriteVarString(buffer, req.Passwd)
	return true
}
func (req *AuthRequestEvent) Decode(buffer *bytes.Buffer) bool {
	var ok bool
	req.Appid, ok = codec.ReadVarString(buffer)
	if !ok {
		return false
	}
	req.User, ok = codec.ReadVarString(buffer)
	if !ok {
		return false
	}
	req.Passwd, ok = codec.ReadVarString(buffer)
	return ok
}
func (req *AuthRequestEvent) GetType() uint32 {
	return AUTH_REQUEST_EVENT_TYPE
}
func (req *AuthRequestEvent) GetVersion() uint32 {
	return 1
}

type AuthResponseEvent struct {
	Appid string
	Token string
	Error string
	HashField
	Attachement
}

func (req *AuthResponseEvent) Encode(buffer *bytes.Buffer) bool {
	codec.WriteVarString(buffer, req.Appid)
	codec.WriteVarString(buffer, req.Token)
	codec.WriteVarString(buffer, req.Error)
	return true
}
func (req *AuthResponseEvent) Decode(buffer *bytes.Buffer) bool {
	var ok bool
	req.Appid, ok = codec.ReadVarString(buffer)
	if !ok {
		return false
	}
	req.Token, ok = codec.ReadVarString(buffer)
	if !ok {
		return false
	}
	req.Error, ok = codec.ReadVarString(buffer)
	return ok
}
func (req *AuthResponseEvent) GetType() uint32 {
	return AUTH_RESPONSE_EVENT_TYPE
}
func (req *AuthResponseEvent) GetVersion() uint32 {
	return 1
}

type HTTPMessageEvent struct {
	Headers vector.Vector
	Content bytes.Buffer
}

func (msg *HTTPMessageEvent) SetHeader(name, value string) {

}

func (msg *HTTPMessageEvent) DoEncode(buffer *bytes.Buffer) bool {
	var slen int = msg.Headers.Len()
	codec.WriteUvarint(buffer, uint64(slen))
	for i := 0; i < slen; i++ {
		header, ok := msg.Headers.At(i).([]string)
		if ok {
			codec.WriteVarString(buffer, header[0])
			codec.WriteVarString(buffer, header[1])
		}
	}
	b := msg.Content.Bytes()
	codec.WriteVarBytes(buffer, b)
	return true
}

func (msg *HTTPMessageEvent) DoDecode(buffer *bytes.Buffer) bool {
	length, err := codec.ReadUvarint(buffer)
	if err != nil {
		return false
	}
	for i := 0; i < int(length); i++ {
		headerName, ok := codec.ReadVarString(buffer)
		if !ok {
			return false
		}
		headerValue, ok := codec.ReadVarString(buffer)
		if !ok {
			return false
		}
		msg.Headers.Push([]string{headerName, headerValue})
	}
	b, ok := codec.ReadVarBytes(buffer)
	if !ok {
		return false
	}
	msg.Content.Write(b)
	return true
}

type HTTPRequestEvent struct {
	HTTPMessageEvent
	Method string
	Url    string
	HashField
	Attachement
}

func (req *HTTPRequestEvent) Encode(buffer *bytes.Buffer) bool {
	codec.WriteVarString(buffer, req.Method)
	codec.WriteVarString(buffer, req.Url)
	req.DoEncode(buffer)
	return true
}
func (req *HTTPRequestEvent) Decode(buffer *bytes.Buffer) bool {
	var ok bool
	req.Method, ok = codec.ReadVarString(buffer)
	if !ok {
		return false
	}
	req.Url, ok = codec.ReadVarString(buffer)
	if !ok {
		return false
	}
	ok = req.DoDecode(buffer)
	if !ok {
		return false
	}
	return true
}
func (req *HTTPRequestEvent) GetType() uint32 {
	return HTTP_REQUEST_EVENT_TYPE
}
func (req *HTTPRequestEvent) GetVersion() uint32 {
	return 1
}

type HTTPResponseEvent struct {
	HTTPMessageEvent
	Status uint32
	HashField
	Attachement
}

func (res *HTTPResponseEvent) Encode(buffer *bytes.Buffer) bool {
	codec.WriteUvarint(buffer, uint64(res.Status))
	res.DoEncode(buffer)
	return true
}
func (res *HTTPResponseEvent) Decode(buffer *bytes.Buffer) bool {
	tmp, err := codec.ReadUvarint(buffer)
	if err != nil {
		return false
	}
	res.Status = uint32(tmp)
	ok := res.DoDecode(buffer)
	if !ok {
		return false
	}
	return true
}

func (req *HTTPResponseEvent) GetType() uint32 {
	return HTTP_RESPONSE_EVENT_TYPE
}
func (req *HTTPResponseEvent) GetVersion() uint32 {
	return 1
}

type SegmentEvent struct {
	sequence uint32
	total    uint32
	content  bytes.Buffer
	HashField
	Attachement
}

func (seg *SegmentEvent) Encode(buffer *bytes.Buffer) bool {
	codec.WriteUvarint(buffer, uint64(seg.sequence))
	codec.WriteUvarint(buffer, uint64(seg.total))
	buffer.Write(seg.content.Bytes())
	return true
}
func (seg *SegmentEvent) Decode(buffer *bytes.Buffer) bool {
	tmp, err := codec.ReadUvarint(buffer)
	if err != nil {
		return false
	}
	seg.sequence = uint32(tmp)
	tmp, err = codec.ReadUvarint(buffer)
	if err != nil {
		return false
	}
	seg.total = uint32(tmp)
	length, err := codec.ReadUvarint(buffer)
	buf := make([]byte, length)
	realLen, err := buffer.Read(buf)
	if err != nil || uint64(realLen) < length {
		return false
	}
	seg.content.Write(buf)
	return true
}

func (seg *SegmentEvent) GetType() uint32 {
	return RESERVED_SEGMENT_EVENT_TYPE
}
func (seg *SegmentEvent) GetVersion() uint32 {
	return 1
}

const (
	C_NONE    uint32 = 0
	C_SNAPPY  uint32 = 1
	C_LZF     uint32 = 2
	C_FASTLZ  uint32 = 3
	C_QUICKLZ uint32 = 4
)

type CompressEvent struct {
	CompressType uint32
	Ev           Event
	HashField
	Attachement
}

func (ev *CompressEvent) Encode(buffer *bytes.Buffer) bool {
	if ev.CompressType != C_NONE && ev.CompressType != C_SNAPPY {
		ev.CompressType = C_NONE
	}
	codec.WriteUvarint(buffer, uint64(ev.CompressType))
	//ev.ev.Encode(buffer);
	var buf bytes.Buffer
	EncodeEvent(&buf, ev.Ev)
	switch ev.CompressType {
	case C_NONE:
		{
			buffer.Write(buf.Bytes())
		}
	case C_SNAPPY:
		{
			evbuf := make([]byte, 0)
			newbuf := snappy.Encode(evbuf, buf.Bytes())
			buffer.Write(newbuf)
		}
	}
	return true
}
func (ev *CompressEvent) Decode(buffer *bytes.Buffer) bool {
	tmp, err := codec.ReadUvarint(buffer)
	if err != nil {
		return false
	}
	ev.CompressType = uint32(tmp)
	var success bool
	switch ev.CompressType {
	case C_NONE:
		{
			success, ev.Ev = ParseEvent(buffer)
			return success
		}
	case C_SNAPPY:
		{
            b := make([]byte, 0, 0)
			newbuf, ok, _ := snappy.Decode(b,buffer.Bytes())
			if !ok {
				return false
			}
			success, ev.Ev = ParseEvent(bytes.NewBuffer(newbuf))
			return success
		}
	}
	return true
}

func (ev *CompressEvent) GetType() uint32 {
	return COMPRESS_EVENT_TYPE
}
func (ev *CompressEvent) GetVersion() uint32 {
	return 1
}

const (
	E_NONE uint32 = 0
	E_SE1  uint32 = 1
)

type EncryptEvent struct {
	EncryptType uint32
	Ev          Event
	HashField
	Attachement
}

func (ev *EncryptEvent) Encode(buffer *bytes.Buffer) bool {
	codec.WriteUvarint(buffer, uint64(ev.EncryptType))
	//ev.ev.Encode(buffer);
	var buf bytes.Buffer
	EncodeEvent(&buf, ev.Ev)
	switch ev.EncryptType {
	case E_NONE:
		{
			buffer.Write(buf.Bytes())
		}
	case E_SE1:
		{
			newbuf := se1.Encrypt(&buf)
			buffer.Write(newbuf.Bytes())
		}
	}
	return true
}
func (ev *EncryptEvent) Decode(buffer *bytes.Buffer) bool {
	tmp, err := codec.ReadUvarint(buffer)
	if err != nil {
		return false
	}
	ev.EncryptType = uint32(tmp)
	var success bool
	switch ev.EncryptType {
	case E_NONE:
		{
			success, ev.Ev = ParseEvent(buffer)
			return success
		}
	case E_SE1:
		{
			newbuf := se1.Decrypt(buffer)
			success, ev.Ev = ParseEvent(newbuf)
			return success
		}
	}
	return true
}

func (ev *EncryptEvent) GetType() uint32 {
	return ENCRYPT_EVENT_TYPE
}
func (ev *EncryptEvent) GetVersion() uint32 {
	return 1
}
