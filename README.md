# sope

An interactive terminal UI for browsing and editing the keys inside
[SOPS](https://github.com/getsops/sops)-encrypted dotenv files that use
[age](https://github.com/FiloSottile/age) recipients.

It is meant for the common chore of changing one value in an encrypted
`*.enc.env` file without hand-editing ciphertext: list the files, open one,
unlock it with your age key, and read or change individual keys. The file is
re-encrypted in place, so your normal review/commit flow stays the source of
truth.

For scripted, one-off access use `sops` and `age` directly. `sope` exists for
the interactive browse-and-edit case those binaries do not cover.

## Security model

- No network listener: it is a local terminal program; secrets never traverse a
  socket.
- The age key is entered at runtime, in a masked prompt, and kept in memory only
  for the session. It is never read from a mounted file by default, never
  written to disk, and never placed in the environment. It is gone when the
  process exits.
- Plaintext stays in memory: values are never written to disk in the clear and
  never logged. The encrypted file is rewritten in place.
- Alternate screen: on quit the terminal is restored with no revealed secret
  left in the scrollback.
- Values render masked by default; revealing is per-value and opt-in.

## Install

```sh
go install github.com/eiseron/sope/cmd/sope@latest
```

Or download a prebuilt binary from the Releases page, or run the image:

```sh
docker run --rm -it -v "$PWD:/work" -e SECRETS_ROOT=/work \
  registry.gitlab.com/eiseron/stack/sope:latest
```

## Usage

```sh
SECRETS_ROOT=/path/to/repos sope
```

Files are discovered by matching each `.sops.yaml`'s `creation_rules` under the
root. Open a file, paste your `AGE-SECRET-KEY-1...` into the unlock prompt, and
the keys appear. A second file with the same recipient opens without asking
again.

Keys: `j`/`k` move, `enter` open, `r` reveal the selected value, `esc` go back,
`q` quit.

## Development

The Go toolchain runs through docker compose, so no local Go install is needed:

```sh
make test    # go test ./... -race
make lint    # gofmt + go vet
make build   # build ./bin/sope
make image   # build the docker image
```

## License

[Apache-2.0](./LICENSE).
