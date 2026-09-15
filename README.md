# livediff

Stay on top of what your coding agents are doing with a live diff feed, right in
your terminal.

`livediff` watches the current Git repository and shows every edit as it happens,
whether it comes from an agent, an editor, or another tool. Changes appear like a
regular Git diff: added lines are green, removed lines are red, and unchanged
lines provide context. Each edit is shown once, keeping the feed focused on what
your agents are changing now.

## Install

Run this in your terminal:

```sh
curl -fsSL https://raw.githubusercontent.com/stagas/livediff/main/install.sh | sh
```

The installer supports Linux and macOS on x64 or ARM64, and Windows x64 through
Git Bash. It downloads a standalone executable; only Git needs to be installed.

## Get started

Open a terminal in any Git repository and run:

```sh
livediff
```

Now edit and save a file. You will see output similar to:

```diff
diff --git a/example.ts b/example.ts 3:47:12 PM
--- a/example.ts
+++ b/example.ts
@@ -1,3 +1,3 @@
-const greeting = "Hello";
+const greeting = "Hello, world!";
```

Only edits made after `livediff` starts are displayed. It watches tracked files
and new files that are not excluded by your Git ignore rules.

Press `q` or Ctrl-C to stop.

## Compact mode

For a quieter view with only the filename, time, and changed lines, run:

```sh
livediff --compact
```

The short form works too:

```sh
livediff -c
```

Compact output looks like this:

```diff
example.ts 3:47:12 PM
-const greeting = "Hello";
+const greeting = "Hello, world!";
```

## Color and install location

Disable colors when redirecting output or using a terminal without color support:

```sh
NO_COLOR=1 livediff
```

The installer uses `/usr/local/bin` when it is writable and `~/.local/bin`
otherwise. To choose a different directory:

```sh
curl -fsSL https://raw.githubusercontent.com/stagas/livediff/main/install.sh | INSTALL_DIR="$HOME/bin" sh
```

Release checksums are available as `SHA256SUMS` on the GitHub Releases page.

## Development

The project uses Go. To test or compile it from source:

```sh
go test ./...
go build -o dist/livediff .
```

## License

[MIT](LICENSE) © 2026 stagas
