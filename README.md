# without-clash

An util to bypass clash-premium tun for commands

# Requirement
Kernel Features:
1. `cgroup2`
2. `ebpf` && `cgroup2 sock attach point`
3. `iproute2`

# Install
```bash
make build
sudo make install
sudo systemctl enable --now without-clash-daemon.service
```

# Usage
```bash
without-clash <command...>

# without-clash ping 1.1.1.1
```