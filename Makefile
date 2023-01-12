.PHONY: build install clean

without-clash: $(wildcard *.go) $(wildcard cgroup/*.go) $(wildcard iproute2/*.go)
	CGO_ENABLED=0 go build -o without-clash

build: without-clash

install: build ext/without-clash-daemon.service
	install -D -m 0755 -o root -g root without-clash /usr/bin/without-clash
	install -D -m 0644 -o root -g root ext/without-clash-daemon.service /usr/lib/systemd/system/without-clash-daemon.service

uninstall:
	rm -rf /usr/bin/without-clash
	rm -rf /usr/lib/systemd/system/without-clash-daemon.service

clean:
	rm -rf without-clash