package main

import (
	"encoding"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/mailru/easyjson/jwriter"
)

func main() {
	m := Message{
		ID:     123123,
		Name:   "Blah",
		Folder: FolderID(0),
		Flags: Flags{
			Read: true,
		},
		Dates: []time.Time{
			time.Date(2017, 03, 01, 05, 24, 0, 0, time.UTC),
		},
		Users: []*User{
			{"user name 2", 30, 'm'},
			{"user name 2", 28, 'f'},
		},
	}

	w := Writer{}
	b, _ := w.Encode(m)
	fmt.Printf("%s", b)
}

type FolderID uint64

type Message struct {
	ID     uint64 `json:"id,string"`
	Name   string
	Folder FolderID `json:",string"`
	Flags  Flags
	Dates  []time.Time
	Users  []*User
}

type Flags struct {
	Read    bool `json:",omitempty"`
	Archive bool `json:",omitempty"`
}

type User struct {
	Name string
	Age  uint
	Sex  byte
}

type Writer struct {
	jw jwriter.Writer
}

func (w *Writer) Encode(v interface{}) ([]byte, error) {
	w.jw.RawByte('{')
	writePrefixed(w, "Message", reflect.ValueOf(v))
	w.jw.RawByte('}')
	return w.jw.BuildBytes()
}

func (w *Writer) writeLeaf(name string, v interface{}) {
	if v == nil {
		return
	}

	w.jw.String(name)
	w.jw.RawByte(':')

	switch v := v.(type) {
	case int:
		w.jw.Int(v)
	case uint8:
		w.jw.Uint(uint(v))
	case uint64:
		w.jw.Uint64Str(v)
	case bool:
		w.jw.Bool(v)
	case string:
		w.jw.String(v)
	default:
		if v1, ok := v.(encoding.TextMarshaler); ok {
			w.jw.RawByte('"')
			w.jw.Raw(v1.MarshalText())
			w.jw.RawByte('"')
			return
		}
	}
}

var (
	jsonMarshalerIface = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerIface = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

func writePrefixed(w *Writer, prefix string, val reflect.Value) {
	kind := val.Kind()
	if kind == reflect.Ptr || kind == reflect.Interface {
		val = reflect.Indirect(val)
		kind = val.Kind()
	}

	if !val.IsValid() {
		return
	}

	t := val.Type()

	if reflect.PtrTo(t).Implements(jsonMarshalerIface) || reflect.PtrTo(t).Implements(textMarshalerIface) {
		w.writeLeaf(prefix, val.Interface())
		return
	}

	switch kind {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i += 1 {
			childValue := val.Field(i)
			childKey := t.Field(i).Name
			if i > 0 {
				w.jw.RawByte(',')
			}
			writePrefixed(w, prefix+"_"+childKey, childValue)
		}

	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			if i > 0 {
				w.jw.RawByte(',')
			}
			writePrefixed(w, prefix+"_"+strconv.Itoa(i), val.Index(i))
		}

	case reflect.Int, reflect.Int32, reflect.Int64:
		w.writeLeaf(prefix, val.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		w.writeLeaf(prefix, val.Uint())

	case reflect.Bool:
		w.writeLeaf(prefix, val.Bool())

	case reflect.String:
		w.writeLeaf(prefix, val.String())

	default:
		log.Printf("unknown type: %s", val)
	}
}
