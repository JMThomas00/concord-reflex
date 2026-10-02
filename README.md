# Reflex

How fast are you? Press **Space**, wait, and press it again the moment the
screen says **NOW**. A reaction-time game for the terminal and for
[Concord](https://github.com/JMThomas00/Concord) channels, where each channel
keeps a leaderboard.

In Concord the game runs **on each player's own computer**, as sandboxed
client code, so the timing is exact however far away the server is. It's
also the smallest complete example of a Concord plugin with client code.

## Play it on your own computer

**Download:** from the [Releases](https://github.com/JMThomas00/concord-reflex/releases)
page, get the zip for your system (`concord-reflex_windows_amd64.zip`,
`concord-reflex_darwin_arm64.zip` for Apple silicon, `concord-reflex_linux_amd64.zip`, ...),
unzip it, and run the program inside from a terminal:

```sh
./concord-reflex          # Windows: .\concord-reflex.exe
```

On macOS, if it's blocked as being from an unidentified developer, run
`xattr -d com.apple.quarantine concord-reflex` once. **Or with Go installed:**
`go install github.com/JMThomas00/concord-reflex@latest`, then run `concord-reflex`.

Your best time is kept in your config folder. **q** quits.

## Play it on a Concord server

You need to be the server owner, or have the **Manage Plugins** permission.

1. In Concord, open **Server Settings → Plugins** and press **I** (install).
2. Type `JMThomas00/concord-reflex` and press Enter. Concord downloads the
   latest release for the server's own system, verifies it, and starts it.
3. Open **Server Settings → Channels**, create a channel, and choose
   **Reflex** as its type.
4. Open the channel and press **Tab**. The first time, Concord asks whether
   to run Reflex's code. It shows the publisher key's fingerprint,
   `3413 13a6 8077 8f8d 2b5e dd30 317f 4e7f`, and what the code can do:
   - draw in its pane;
   - keep your best time on your computer;
   - send your times to the server for the leaderboard.

   Press **A** to allow it.

The code runs sandboxed: it can't touch your files, the network or other
programs. Members who don't allow it see a note instead of the game.
**Ctrl+]** gives the keyboard back to Concord.

To update later: select it in **Server Settings → Plugins**, press **U**, then Enter.

## Layout

- `game/`: the game itself (rounds, timing, ratings, drawing, theme colors).
- `clientcode/`: the game as Concord client code (WebAssembly, `sdk/client`).
- `server.go`: the server half, which keeps each channel's leaderboard and
  ignores impossible times (client code runs on players' computers, so what
  it reports can't be trusted).
- `local.go`: the same game standalone, with Bubble Tea.
- `release.go`: `go run release.go` builds and signs the client code, builds
  the server half for every platform, and packs the zips Concord installs.

## Releasing

The client code is signed with the publisher key, which stays on the
publisher's computer. Concord refuses code that isn't signed by the
`publisher_key` in `plugin.toml`. Members' clients remember the key, so a
new one makes everyone agree again.

```sh
git tag v0.2.0 && git push --tags
CONCORD_PUBLISHER_KEY_FILE=path/to/publisher.key go run release.go
gh release create v0.2.0 dist/*.zip --title v0.2.0 --notes "..."
```
