package planb

import (
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
		w.AppendInt(int64(v))
	case string:
		w.AppendBulkString(v)
	case []byte:
		w.AppendBulk(v)
	case resp.CommandArgument:
		w.AppendBulk([]byte(v))
	case float32:
		w.AppendInlineString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case float64:
		w.AppendInlineString(strconv.FormatFloat(float64(v), 'f', -1, 64))
	case []string:
		w.AppendArrayLen(len(v))
		for _, s := range v {
			w.AppendBulkString(s)
		}
	case [][]byte:
		w.AppendArrayLen(len(v))
		for _, b := range v {
			w.AppendBulk(b)
		}
	case []int:
		w.AppendArrayLen(len(v))
		for _, n := range v {
			w.AppendInt(int64(n))
		}
	case []int64:
		w.AppendArrayLen(len(v))
		for _, n := range v {
			w.AppendInt(n)
		}
	case map[string]string:
		w.AppendArrayLen(len(v) * 2)
		for k, s := range v {
			w.AppendBulkString(k)
			w.AppendBulkString(s)
		}
	case CustomResponse:
		v.AppendTo(w)
	default:
		w.AppendErrorf("ERR unsupported response type %T", v)
	}
}
