# Development scripts

Служебные scripts для local build/check.

Основные команды:

```sh
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
./scripts/test-archive.sh
```

`test-archive.sh` проверяет, что embedded archive содержит top-level directory
`neto/`, installer/uninstaller, LuCI files и binaries layout.
