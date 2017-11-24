package planb_test

import (
	"fmt"
	"testing"

	"github.com/bsm/planb"
	"github.com/bsm/redeo/resp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("SubCommands", func() {

	subject := planb.SubCommands{
		"foo": planb.HandlerFunc(func(cmd *planb.Command) interface{} { return "bar" }),
		"baz": planb.HandlerFunc(func(cmd *planb.Command) interface{} {
			if len(cmd.Args) == 0 {
				return fmt.Errorf("failed to execute %q", cmd.Name)
			}
			return []interface{}{cmd.Args[0].String(), 0.75}
		}),
	}

	DescribeTable("success",
		func(cmd *planb.Command, exp interface{}) {
			Expect(subject.ServeRequest(cmd)).To(Equal(exp))
		},

		Entry("simple", &planb.Command{Name: "custom", Args: []resp.CommandArgument{
			resp.CommandArgument("foo"),
		}}, "bar"),
		Entry("more complex", &planb.Command{Name: "custom", Args: []resp.CommandArgument{
			resp.CommandArgument("baz"),
			resp.CommandArgument("echo"),
		}}, []interface{}{"echo", 0.75}),
	)

	DescribeTable("failure",
		func(cmd *planb.Command, exp string) {
			Expect(subject.ServeRequest(cmd)).To(MatchError(exp))
		},

		Entry("no sub-command",
			&planb.Command{Name: "custom"},
			`wrong number of arguments for 'custom'`),
		Entry("bad sub-command",
			&planb.Command{Name: "custom", Args: []resp.CommandArgument{
				resp.CommandArgument("bad"),
			}},
			`Unknown custom subcommand 'bad'`),
		Entry("invalid sub-command",
			&planb.Command{Name: "custom", Args: []resp.CommandArgument{
				resp.CommandArgument("baz"),
			}},
			`failed to execute "custom baz"`),
	)

})

// --------------------------------------------------------------------

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "planb")
}
