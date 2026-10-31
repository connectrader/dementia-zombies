# Dementia Zombies

An infinite top-down zombie survival game. Every kill makes you stronger; every second outside makes you forget. The safehouse keeps reappearing near you, dementia improves your loot and toughens the dead, and the numbers never stop climbing. Up to 4 players over LAN.

## Play

Download the latest **Dementia-Zombies-vX.Y.Z-Windows.zip** from [Releases](../../releases), unzip it, and run `Dementia Zombies.exe`. The game updates itself from here after that (pause menu, "Check for updates").

## Layout

| Path | What it is |
| --- | --- |
| `dementia-zombies.html` | The whole game, one file. `GAME_VERSION` at the top of its script is the single version number. |
| `dz-desktop/` | The Windows app: a small Go server that serves the game into an Edge app window, relays LAN co-op, keeps saves on disk and self-updates. |
| `dz-desktop/build.py` | Builds the .exe and the versioned packages (`--bump major|minor|patch` to raise the version first). |
| `dz-desktop/release.py` | Builds, commits, pushes and publishes a GitHub release that every copy of the game updates from. |
| `CHANGELOG.md` | Release notes. Each release needs a `## vX.Y.Z` section; it becomes the release text and the in-game update notes. |

## Build

Needs Go 1.24+, Python 3 with Pillow, and `llvm-windres` (for the .exe icon and version info).

```
cd dz-desktop
python3 build.py
```

## Release

```
python3 dz-desktop/release.py --bump minor
```

Players' copies look for the latest release's `DementiaZombies-vX.Y.Z-windows-x64.exe` and `SHA256SUMS.txt`, and refuse anything that doesn't match the checksum.

Made by connectrader, LLC.
