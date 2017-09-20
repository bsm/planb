default: vet test

test:
	go test .

vet:
	go vet .

bench:
	go test . -run=NONE -bench=. -benchmem -benchtime=5s

doc: README.md

README.md: README.md.tpl $(wildcard *.go)
	becca -package .
