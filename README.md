# Reflex

How fast are you? A quick-draw duel at high noon: press **Space** when you're
ready, keep your hands still through **WAIT...**, and draw the moment it says
**DRAW!** For the terminal and for [Concord](https://github.com/JMThomas00/Concord)
channels, where it's a little arcade cabinet: two gunslingers, a tumbleweed,
seven-segment times, titles from GREENHORN to LEGEND, a Hall of Fame, and
**Bounties** to spend on hats and backdrops at the **General Store**.

In Concord the game runs **on each player's own computer**, as sandboxed
client code, so the timing is exact however far away the server is. It's
also a complete example of a Concord plugin with client code (the SDK's
`examples/reflex` is the smallest one).

## The game

- **QUICK DRAW** `◂ NORMAL ▸`: beat the outlaw. He draws in about 430 ms
  (easy), 300 (normal) or 215 (hard).
- **SHOWDOWN**: challenge someone in the channel. You both see the word at the
  same moment, and each of you is timed on your own computer, so lag can't cost
  anyone; the faster draw wins the round.
- **PRACTICE**: just you and the clock.
- **GENERAL STORE**: 9 hats (stetson and black hat to start ... up to a grape
  head and a robot) and 7 backdrops (high noon, sunset, midnight, snowed in, red
  canyon, the vineyard, the Moon). You earn a **Bounty** for each new
  achievement, every three duels won in a row and every ten duels, and trade one
  for a pick of three.
- **HALL OF FAME**, **HOW TO PLAY** and **OPTIONS** (sound, effects).

Draw before the word, or within a tenth of a second of it (that's a guess), and
it's **TOO SOON!**: a foul, and the duel goes to the other side. Your title
comes from your time: under 300 ms RANGER, 250 DEPUTY, 210 SHERIFF, 170 LEGEND.

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

Standalone it's the same arcade, without showdowns; your record, Bounties and
gear are kept in your config folder. **Esc** on the title screen quits.

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
   - play sounds;
   - talk to the server half (your times, your gear, showdowns).

   Press **A** to allow it.

The code runs sandboxed: it can't touch your files, the network or other
programs. Members who don't allow it see a note instead of the game.
**Esc** goes back a screen; on the title screen it gives the keyboard back to
Concord. A pane smaller than 64×24 gets a plain practice screen.

Updating from 0.1 asks everyone again once, because the code now plays sounds
(and no longer keeps anything on your computer: your record lives on the
server).

To update later: select it in **Server Settings → Plugins**, press **U**, then Enter.

## Layout

- `game/`: the arcade itself (`app.go`: screens, keys and the duel;
  `draw.go`: drawing with the SDK's `arcade` canvas; `art.go`: gunslingers and
  backdrops; `catalog.go`: gear, titles, levels; `msg.go`: what the two halves
  say to each other; `sounds.go`).
- `clientcode/`: the arcade as Concord client code (WebAssembly, `sdk/client`).
- `server.go`: the server half: records, achievements and the leaderboard,
  Bounties and gear, the Hall of Fame, and showdowns, which it judges itself
  from both players' times. It ignores impossible times (client code runs on
  players' computers, so what it reports can't be fully trusted).
- `local.go`: the same arcade standalone, with Bubble Tea and the server half
  in the same program.
- `release.go`: `go run release.go` builds and signs the client code, builds
  the server half for every platform, and packs the zips Concord installs
  (with the sounds).

## Releasing

The client code is signed with the publisher key, which stays on the
publisher's computer. Concord refuses code that isn't signed by the
`publisher_key` in `plugin.toml`. Members' clients remember the key, so a
new one makes everyone agree again.

```sh
git tag v0.3.0 && git push --tags
CONCORD_PUBLISHER_KEY_FILE=path/to/publisher.key go run release.go
gh release create v0.3.0 dist/*.zip --title v0.3.0 --notes "..."
```

## License

MIT License — see LICENSE file for details.
