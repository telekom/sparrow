<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Testing the traceroute check

This guide walks you through running sparrow's traceroute
check in a simulated multi-hop network using
[Kathara](https://github.com/KatharaFramework/Kathara), a
container-based network emulation tool.

The lab files in [`pkg/checks/traceroute/test-lab/`][test-lab]
configure a small topology with a client, two routers, and a
webserver — enough hops to exercise the traceroute logic
end-to-end.

[test-lab]: pkg/checks/traceroute/test-lab

## Prerequisites

- [Kathara](https://github.com/KatharaFramework/Kathara)
- Wireshark (optional, for packet inspection)

## Start the Lab Network

From the test-lab directory:

```bash
kathara lstart --noterminals
```

This boots the
[static-routing topology][topology]. Omit `--noterminals` if
you want a terminal window per container.

[topology]: https://github.com/KatharaFramework/Kathara-Labs/blob/main/main-labs/basic-topics/static-routing/004-kathara-lab_static-routing.pdf

## Connect to the Client

Open a separate terminal:

```bash
kathara connect pc1
```

## Explore the Network (Optional)

The lab contains two routers between the client and the
webserver at `200.1.1.7`:

```bash
export WEBSERVER=200.1.1.7
traceroute $WEBSERVER
```

```text
traceroute to 200.1.1.7, 30 hops max, 60 byte packets
 1  195.11.14.1  0.972 ms  1.093 ms  1.095 ms
 2  100.0.0.10   1.543 ms  1.712 ms  1.838 ms
 3  200.1.1.7    2.232 ms  2.310 ms  2.394 ms
```

Verify the webserver responds:

```bash
curl $WEBSERVER
```

## Build and Run Sparrow

Kathara mounts `test-lab/shared/` into every container, so
you can build on the host and run inside the lab without
a custom image.

On your host:

```bash
go build -o sparrow . && \
  mv sparrow pkg/checks/traceroute/test-lab/shared/
```

Inside the client container, create a config and run:

```bash
cd /shared
cat > config.yaml <<'EOF'
name: sparrow.dev
loader:
  type: file
  interval: 30s
  file:
    path: ./config.yaml
traceroute:
  interval: 5s
  timeout: 3s
  retries: 3
  maxHops: 8
  targets:
    - addr: 200.1.1.7
      port: 80
EOF

./sparrow run --config config.yaml
```

## Debug with Tcpdump and Wireshark

The container includes `tcpdump` for low-level packet
inspection:

```bash
tcpdump -w /shared/dump.pcap
```

Then open the capture on your host:

```bash
wireshark -r pkg/checks/traceroute/test-lab/shared/dump.pcap
```

## Clean Up

```bash
kathara lclean
```

## See Also

- [Traceroute check](../checks/traceroute.md)
- [Developer guide](README.md)
