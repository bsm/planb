package planb_test

import (
	"bytes"

	"github.com/bsm/planb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InmemStore", func() {
	var subject *planb.InmemStore

	BeforeEach(func() {
		subject = planb.NewInmemStore()
		Expect(subject.Put([]byte("key1"), []byte("val1"))).To(Succeed())
		Expect(subject.Put([]byte("key2"), []byte("val2"))).To(Succeed())
		Expect(subject.Put([]byte("key3"), []byte("val3"))).To(Succeed())
		Expect(subject.Put([]byte("key4"), []byte("val4"))).To(Succeed())
	})

	It("should GET", func() {
		Expect(subject.Get([]byte("key5"))).To(BeNil())
		Expect(subject.Get([]byte("key2"))).To(Equal([]byte("val2")))
	})

	It("should PUT", func() {
		Expect(subject.Get([]byte("key2"))).To(Equal([]byte("val2")))
		Expect(subject.Delete([]byte("key2"))).To(Succeed())
		Expect(subject.Get([]byte("key2"))).To(BeNil())
	})

	It("should snapshot/restore", func() {
		buf := new(bytes.Buffer)
		Expect(subject.Snapshot(buf)).To(Succeed())
		Expect(buf.Len()).To(BeNumerically("~", 40, 10))

		Expect(subject.Delete([]byte("key1"))).To(Succeed())
		Expect(subject.Delete([]byte("key3"))).To(Succeed())
		Expect(subject.Get([]byte("key3"))).To(BeNil())

		Expect(subject.Restore(buf)).To(Succeed())
		Expect(subject.Get([]byte("key3"))).To(Equal([]byte("val3")))
	})

})
