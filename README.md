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
go install github.com/eiseron/sope@latest
```

Or download a prebuilt binary from the Releases page, or run the image:

```sh
docker run --rm -it -v "$PWD:/work" -w /work \
  registry.gitlab.com/eiseron/stack/sope:latest
```

## Usage

By default `sope` searches the current directory:

```sh
cd /path/to/repos && sope
```

You can point it elsewhere with a positional path, or with `SECRETS_ROOT`:

```sh
sope /path/to/repos
SECRETS_ROOT=/path/to/repos sope
```

Files are discovered by matching each `.sops.yaml`'s `creation_rules` under the
root. Open a file, paste your `AGE-SECRET-KEY-1...` into the unlock prompt, and
the keys appear. A second file with the same recipient opens without asking
again.

Press `n` in the file list to create a new file: type a name (the `.enc.env`
extension is added automatically) and it is encrypted to the recipient
configured in the matching `.sops.yaml` rule, then opened so you can add keys.

If there is no `.sops.yaml` yet, that first `n` bootstraps one: `sope` generates
a fresh age key, shows the secret key once for you to save, and on confirmation
writes `.sops.yaml`, creates the file, and opens it. The secret is shown a
single time and never written to disk, so copy it before confirming. There is
no separate `init` command; the first file sets everything up.

File list: `j`/`k` move, `enter` open, `n` new, `q` quit.

Inside a file: `j`/`k` move, `r` reveal, `e` edit, `a` add, `d` delete,
`s` shell, `esc` go back, `q` quit.

## Shell with the secrets loaded

From an open file, `s` drops you into a subshell with that file's keys exported
as environment variables. It runs `$SHELL` (override with `SOPE_SHELL`, falling
back to `/bin/sh`). The variables live only in that subprocess, on top of your
normal environment; when the shell exits you return to `sope` and they are gone.
They are never set in `sope`'s own environment.

On entry it prints a banner, and `SOPE_FILE` holds the loaded file's path so you
know you are inside a `sope` shell and which file it carries.

```sh
# sope: ops/secrets.enc.env loaded, exit to return
echo "$SOPE_FILE"     # ops/secrets.enc.env
terraform plan        # TF_VAR_* are already set
echo "$DATABASE_URL"  # available here only
exit                  # back to sope; the variables are gone
```

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
