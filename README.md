# tossit

File transfer tool. Send from CLI, receive anywhere.

## Install

```
curl -fsSL https://tossit.dev/install.sh | sh
```

Or build from source:

```
go install ./cmd/tossit
```

## Usage

```
tossit send file.zip          # gives you a code + URL
tossit receive abc-xyz        # download with code
```

Sender uses CLI, receiver can use CLI or just open the link in a browser. No install needed on the other end.

## How it works

Files are streamed through a relay server, encrypted end-to-end. The relay never sees your data.

A free public relay is available at `relay.tossit.dev`. You can also self-host:

```
tossit relay --port 8080
```

## Self-host

Run your own relay server:

```
go install ./cmd/relay
relay --port 8080
```

Or with Docker:

```
docker run -p 8080:8080 ghcr.io/dilgo-dev/tossit-relay
```

## Contact

contact@tossit.dev

## License

MIT
