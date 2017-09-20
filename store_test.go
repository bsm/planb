package planb_test

import (
	. "github.com/onsi/ginkgo"
	_ "github.com/onsi/gomega"
)

var _ = Describe("KVStore", func() {

	/*
		var subject planb.KVStore
		var dir string

		BeforeEach(func() {
			var err error

			dir, err = ioutil.TempDir("", "planb")
			Expect(err).NotTo(HaveOccurred())

			subject, err = planb.OpenKV(dir)
			Expect(err).NotTo(HaveOccurred())

			Expect(subject.Put([]byte("key1"), []byte("val1"))).To(Succeed())
			Expect(subject.Put([]byte("key2"), []byte("val2"))).To(Succeed())
			Expect(subject.Put([]byte("key3"), []byte("val3"))).To(Succeed())
			Expect(subject.Put([]byte("key4"), []byte("val4"))).To(Succeed())
		})

		AfterEach(func() {
			Expect(subject.Close()).To(Succeed())
			Expect(os.RemoveAll(dir)).To(Succeed())
		})

		It("should GET/PUT/DEL", func() {
			Expect(subject.Get([]byte("key5"))).To(BeNil())
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
	*/

})
