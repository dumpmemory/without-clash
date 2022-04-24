.PHONY: build install clean

without-clash: $(wildcard *.go) $(wildcard cgroup/*.go) $(wildcard iproute2/*.go)
	CGO_ENABLED=0 go build -o without-clash

build: without-clash

install: build
	install -D -m 0755 -o root -g root without-clash /usr/lib/without-clash/without-clash
	install -D -m 0644 -o root -g root ext/without-clash-daemon.service /usr/lib/systemd/system/without-clash-daemon.service
	ln -sf /usr/lib/without-clash/without-clash /usr/bin/without-clash
	ln -sf /usr/lib/without-clash/without-clash /usr/lib/without-clash/without-clash-daemon

uninstall:
	rm -rf /usr/lib/without-clash
	rm -rf /usr/lib/systemd/system/without-clash-daemon.service

clean:
	rm -rf without-clash