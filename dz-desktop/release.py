#!/usr/bin/env python3
"""Cut a release: build locally as a check, commit, push, and push the version tag.
GitHub Actions (.github/workflows/release.yml) then builds the Windows app and publishes the release
that every copy of the game updates from.

Run from the repo (dz-desktop/release.py):
  python3 dz-desktop/release.py              release the version in GAME_VERSION (must have a CHANGELOG.md entry)
  python3 dz-desktop/release.py --bump minor bump first (major / minor / patch), then release

The release gets three files: the self-updater's DementiaZombies-vX.Y.Z-windows-x64.exe, the player-facing
Dementia-Zombies-vX.Y.Z-Windows.zip, and SHA256SUMS.txt (the updater refuses anything that doesn't match).
"""
import os, re, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)
GAME = os.path.join(REPO, 'dementia-zombies.html')
GH = 'connectrader/dementia-zombies'

def run(*a, **k):
    print('$', ' '.join(a))
    return subprocess.run(a, check=True, cwd=k.get('cwd', REPO), text=True, capture_output=k.get('capture', False))

args = [sys.executable, os.path.join(HERE, 'build.py')] + (['--bump', sys.argv[sys.argv.index('--bump') + 1]] if '--bump' in sys.argv else [])
subprocess.run(args, check=True, cwd=HERE)
ver = re.search(r"const GAME_VERSION='([\d.]+)';", open(GAME, encoding='utf-8').read()).group(1)
tag = 'v' + ver

log = open(os.path.join(REPO, 'CHANGELOG.md'), encoding='utf-8').read()
m = re.search(r'^## ' + re.escape(tag) + r'\b.*?\n(.*?)(?=^## v|\Z)', log, re.S | re.M)
if not m:
    sys.exit(f'CHANGELOG.md has no "## {tag}" section. Write the release notes first.')
notes = m.group(1).strip()

rel = os.path.join(HERE, 'dist', 'release')
files = [os.path.join(rel, f) for f in sorted(os.listdir(rel))]

run('git', 'add', '-A')
if subprocess.run(['git', 'diff', '--cached', '--quiet'], cwd=REPO).returncode:
    run('git', 'commit', '-m', f'Release {tag}')
run('git', 'push', 'origin', 'HEAD')
run('git', 'tag', '-a', tag, '-m', f'Dementia Zombies {tag}')
run('git', 'push', 'origin', tag)
print(f'pushed {tag}. GitHub Actions is building and publishing it: https://github.com/{GH}/actions')
