package planb_test

import (
	"bytes"

	"github.com/bsm/planb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InMemStore", func() {
	var subject planb.KVStore

	BeforeEach(func() {
		subject = planb.NewInmemStore()

		txn, err := subject.Begin(true)
		Expect(err).NotTo(HaveOccurred())

		Expect(txn.Put([]byte("key1"), []byte("val1"))).To(Succeed())
		Expect(txn.Put([]byte("key2"), []byte("val2"))).To(Succeed())
		Expect(txn.Put([]byte("key3"), []byte("val3"))).To(Succeed())
		Expect(txn.Put([]byte("key4"), []byte("val4"))).To(Succeed())
		Expect(txn.Commit()).To(Succeed())
	})

	AfterEach(func() {
		Expect(subject.Close()).To(Succeed())
	})

	It("should GET", func() {
		txn, err := subject.Begin(false)
		Expect(err).NotTo(HaveOccurred())
		defer txn.Rollback()

		Expect(txn.Get([]byte("key5"))).To(BeNil())
		Expect(txn.Get([]byte("key2"))).To(Equal([]byte("val2")))
	})

	It("should PUT", func() {
		txn, err := subject.Begin(true)
		Expect(err).NotTo(HaveOccurred())

		Expect(txn.Delete([]byte("key2"))).To(Succeed())
		Expect(txn.Get([]byte("key2"))).To(Equal([]byte("val2")))
		Expect(txn.Commit()).To(Succeed())
		Expect(txn.Get([]byte("key2"))).To(BeNil())
	})

	It("should snapshot/restore", func() {
		buf := new(bytes.Buffer)
		Expect(subject.Snapshot(buf)).To(Succeed())
		Expect(buf.Len()).To(BeNumerically("~", 40, 10))

		txn, err := subject.Begin(true)
		Expect(err).NotTo(HaveOccurred())
		Expect(txn.Delete([]byte("key1"))).To(Succeed())
		Expect(txn.Delete([]byte("key3"))).To(Succeed())
		Expect(txn.Commit()).To(Succeed())
		Expect(txn.Get([]byte("key3"))).To(BeNil())

		Expect(subject.Restore(buf)).To(Succeed())
		Expect(txn.Get([]byte("key3"))).To(Equal([]byte("val3")))
	})

})
