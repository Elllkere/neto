# Release Process

Этот документ описывает, как выпускать новый GitHub Release для `neto`.

Версия `netod` не хранится в отдельном Go файле. При сборке archive
`embedded/pack.sh` берет version из git tag через:

```sh
git describe --tags --always --dirty
```

и подставляет ее в binary через Go linker:

```sh
-ldflags "-X main.version=$VERSION"
```

Поэтому normal release flow - это git tag + archive upload.

## 1. Prepare Worktree

Проверить, что нет незакоммиченных изменений:

```sh
git status
```

Если изменения есть:

```sh
git add .
git commit -m "Prepare release v1.0.0"
```

## 2. Create Tag

Для release `v1.0.0`:

```sh
git tag v1.0.0
```

Для следующего release меняешь tag:

```sh
git tag v1.0.1
```

Проверить:

```sh
git tag --list --sort=-version:refname | head
git describe --tags --always --dirty
```

## 3. Build Archive

```sh
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
./scripts/test-archive.sh
```

Archive будет здесь:

```text
dist/neto-openwrt-embedded.tar.gz
```

Проверить version внутри binary:

```sh
tmp="$(mktemp -d)"
tar -xzf dist/neto-openwrt-embedded.tar.gz -C "$tmp"
"$tmp/neto/bin/linux-amd64/netod" version
rm -rf "$tmp"
```

Expected:

```text
netod v1.0.0
```

## 4. Push To GitHub

```sh
git push origin main
git push origin v1.0.0
```

Или все tags:

```sh
git push origin --tags
```

## 5. Publish GitHub Release

На GitHub:

1. Open repository.
2. Go to Releases.
3. Draft a new release.
4. Select tag `v1.0.0`.
5. Release title: `v1.0.0`.
6. Upload asset:

```text
neto-openwrt-embedded.tar.gz
```

7. Publish release.

Installer downloads archive from:

```text
https://github.com/elllkere/neto/releases/latest/download/neto-openwrt-embedded.tar.gz
```

Поэтому asset name должен быть точно:

```text
neto-openwrt-embedded.tar.gz
```

## Manual Version Override

Если нужно собрать archive без git tag:

```sh
NETO_VERSION=v1.0.0 GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
```

Но для public release лучше использовать git tag, чтобы GitHub Release,
`git describe` и `netod version` совпадали.

## Current Release

Current initial release tag:

```text
v1.0.0
```
