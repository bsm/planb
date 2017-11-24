package planb

import (
	"reflect"
	"strconv"

	"github.com/bsm/redeo/resp"
)

// CustomResponse is a supported response type which can be returned by handlers.
// See CustomResponseFunc for an example.
type CustomResponse interface {
	// AppendTo must be implemented by custom response types
	AppendTo(w resp.ResponseWriter)
}

// CustomResponseFunc wraps custom responses into a functional clause
type CustomResponseFunc func(w resp.ResponseWriter)

// AppendTo implements CustomResponse
func (f CustomResponseFunc) AppendTo(w resp.ResponseWriter) { f(w) }

func respondWith(w resp.ResponseWriter, v interface{}) {
	switch v := v.(type) {
	case nil:
		w.AppendNil()
	case CustomResponse:
		v.AppendTo(w)
	case error:
		w.AppendError("ERR " + v.Error())
	case bool:
		if v {
			w.AppendInt(1)
		} else {
			w.AppendInt(0)
		}
	case int:
		w.AppendInt(int64(v))
	case int8:
		w.AppendInt(int64(v))
	case int16:
		w.AppendInt(int64(v))
	case int32:
		w.AppendInt(int64(v))
	case int64:
		w.AppendInt(v)
	case string:
		w.AppendBulkString(v)
	case []byte:
		w.AppendBulk(v)
	case resp.CommandArgument:
		w.AppendBulk([]byte(v))
	case float32:
		w.AppendInlineString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case float64:
		w.AppendInlineString(strconv.FormatFloat(v, 'f', -1, 64))
	default:
		switch reflect.TypeOf(v).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(v)

			w.AppendArrayLen(s.Len())
			for i := 0; i < s.Len(); i++ {
				respondWith(w, s.Index(i).Interface())
			}
		case reflect.Map:
			s := reflect.ValueOf(v)

			w.AppendArrayLen(s.Len() * 2)
			for _, key := range s.MapKeys() {
				respondWith(w, key.Interface())
				respondWith(w, s.MapIndex(key).Interface())
			}

		default:
			w.AppendErrorf("ERR unsupported response type %T", v)
		}
	}
}
