# Managed binary slots

`embedded/pack.sh` writes `netod` binaries under:

- `linux-amd64/`
- `linux-arm64/`
- `linux-armv7/`
- `linux-mips-softfloat/`
- `linux-mipsle-softfloat/`

Optional managed `sing-box` binaries can be placed in the same directories
before packing. They are copied into the embedded archive and installed to
`/usr/libexec/neto/sing-box` only when the system `sing-box` is missing or
incompatible.

