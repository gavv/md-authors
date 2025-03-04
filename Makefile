all: build

build: tidy
	go build .

install: tidy
	go install -v .

tidy:
	go mod tidy -v

fmt:
	gofmt -s -w src

md:
	markdown-toc --maxdepth 3 --bullets=- -i README.md
	md-authors -f modern -a AUTHORS.md
