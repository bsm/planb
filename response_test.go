package planb_test

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/bsm/planb"
	"github.com/bsm/redeo/resp"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("respondWith",
	func(v interface{}, exp string) {
		b := new(bytes.Buffer)
		w := resp.NewResponseWriter(b)
		planb.RespondWith(w, v)
		Expect(w.Flush()).To(Succeed())
		Expect(strconv.Quote(b.String())).To(Equal(strconv.Quote(exp)))
	},

	Entry("nil", nil, "$-1\r\n"),
	Entry("error", errors.New("failed"), "-ERR failed\r\n"),
	Entry("int", 33, ":33\r\n"),
	Entry("int64", int64(33), ":33\r\n"),
	Entry("bool (true)", true, ":1\r\n"),
	Entry("bool (false)", false, ":0\r\n"),
	Entry("float32", float32(0.1231), "+0.1231\r\n"),
	Entry("float64", 0.7357, "+0.7357\r\n"),
	Entry("string", "many words", "$10\r\nmany words\r\n"),
	Entry("[]byte", []byte("many words"), "$10\r\nmany words\r\n"),
	Entry("[]string", []string{"a", "b", "c"}, "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"),
	Entry("[][]byte", [][]byte{{'a'}, {'b'}, {'c'}}, "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"),
	Entry("[]int", []int{3, 5, 2}, "*3\r\n:3\r\n:5\r\n:2\r\n"),
	Entry("[]int64", []int64{7, 8, 3}, "*3\r\n:7\r\n:8\r\n:3\r\n"),
	Entry("map[string]string", map[string]string{"a": "b"}, "*2\r\n$1\r\na\r\n$1\r\nb\r\n"),
	Entry("custom response", &customResponse{Host: "foo", Port: 8888}, "$17\r\ncustom 'foo:8888'\r\n"),
	Entry("custom error", customError("bar"), "-WRONG bar\r\n"),

	Entry("not supported", uint(1<<63+7), "-ERR unsupported response type uint\r\n"),
)

type customResponse struct {
	Host string
	Port int
}

func (r *customResponse) AppendTo(w resp.ResponseWriter) {
	w.AppendBulkString(fmt.Sprintf("custom '%s:%d'", r.Host, r.Port))
}

type customError string

func (r customError) AppendTo(w resp.ResponseWriter) {
	w.AppendError(r.Error())
}

func (r customError) Error() string {
	return "WRONG " + string(r)
}
