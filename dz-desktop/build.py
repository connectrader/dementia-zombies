#!/usr/bin/env python3
"""Build the desktop edition: patch the web game for LAN co-op, compile Dementia Zombies.exe, and package it.

The version lives in one place: GAME_VERSION at the top of the game's script in dementia-zombies.html.
  python3 build.py              build and package the current version
  python3 build.py --bump minor bump 1.1.0 -> 1.2.0 (or major / patch) in the game file, then build
Packages land in dist/ as "Dementia Zombies v<version> (Windows).zip" and "dementia-zombies-v<version>-source.zip".
"""
import base64, os, re, subprocess, sys, shutil, zipfile

ROOT = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(os.path.dirname(ROOT), 'dementia-zombies.html')
OUT = os.path.join(ROOT, 'dist')
VRX = re.compile(r"const GAME_VERSION='(\d+)\.(\d+)\.(\d+)';")

s = open(SRC, encoding='utf-8').read()
m = VRX.search(s)
assert m, 'GAME_VERSION not found in the game file'
ver = [int(x) for x in m.groups()]
if '--bump' in sys.argv:
    part = sys.argv[sys.argv.index('--bump') + 1]
    i = {'major': 0, 'minor': 1, 'patch': 2}[part]
    ver[i] += 1
    for j in range(i + 1, 3):
        ver[j] = 0
    new = '.'.join(map(str, ver))
    s = VRX.sub(f"const GAME_VERSION='{new}';", s, count=1)
    s = re.sub(r'<span class="ver">v[\d.]+</span>', f'<span class="ver">v{new}</span>', s)
    open(SRC, 'w', encoding='utf-8').write(s)
    print('bumped to', new)
VERSION = '.'.join(map(str, ver))

def rep(a, b, n=1):
    global s
    c = s.count(a)
    assert c == n, f'expected {n} of {a[:70]!r}, found {c}'
    s = s.replace(a, b)

fav = base64.b64encode(open(os.path.join(ROOT, 'favicon.png'), 'rb').read()).decode()
shim = open(os.path.join(ROOT, 'shim.js'), encoding='utf-8').read()
rep('<title>Dementia Zombies</title>',
    '<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">'
    '<title>Dementia Zombies</title>'
    f'<link rel="icon" type="image/png" href="data:image/png;base64,{fav}">'
    '<style>html,body{margin:0;height:100%;background:#15171b;overflow:hidden}</style>'
    f'<script>{shim}</script>')

# co-op wording: the local network instead of claude.ai
rep("It runs through claude.ai's live room, so friends need this page shared with them.",
    "It runs over your local network: anyone on the same Wi-Fi or LAN running Dementia Zombies sees your game in their Co-op list.")
rep("toast('Co-op needs this page open on claude.ai while signed in.','bad')",
    "toast('Co-op could not reach the game\\'s network service. Restart Dementia Zombies.','bad')", 2)
rep("""Co-op runs through claude.ai's live room, so it needs this page open on claude.ai while you're signed in. Friends need access to this page too: share it with them from the page's Share menu.""",
    """Co-op could not reach the game's network service. Close and reopen Dementia Zombies.""")
rep("Friends with this page open can join from Co-op (C).", "Anyone on your network can join from Co-op (C).")
rep("""<p class="fine">Friends open this page, press C (or Co-op), and join this code or pick your game from the list. ${4-NET.slots.length}""",
    """<p class="fine">Anyone on your network opens Dementia Zombies, presses C and picks your game from the list, or types this code.${window.__dzLan?` If your game doesn't show up for them, they can type your address, <b>${esc(window.__dzLan)}</b>. A friend without the app can open <b>http://${esc(window.__dzLan.includes(':')?window.__dzLan:window.__dzLan+':47800')}</b> in a browser.`:''} ${4-NET.slots.length}""")
rep("""<input id="coCode" class="coin" maxlength="8" placeholder="Code" aria-label="Game code"><button type="button" class="btn alt" data-co="join">Join</button></div>""",
    """<input id="coCode" class="coin" maxlength="8" placeholder="Code" aria-label="Game code"><button type="button" class="btn alt" data-co="join">Join</button></div><div class="cojoin" style="margin-top:8px"><input id="coAddr" class="coin" maxlength="60" placeholder="Host address, like 192.168.1.20" aria-label="Host address"><button type="button" class="btn alt" data-co="addr">Find</button></div>""")
rep("""'<p class="fine" style="margin:8px 0 0">No open games on this page right now.</p>'""",
    """'<p class="fine" style="margin:8px 0 0">Looking for games on your network… Hosts show up here within a few seconds. If one doesn\\'t, type the host\\'s address (it\\'s on their Co-op screen) and press Find. If nobody can see anyone, allow Dementia Zombies through Windows Firewall on private networks.</p>'""")
rep("""else if(a==='join')joinGame(($('coCode')||{}).value);""",
    """else if(a==='join')joinGame(($('coCode')||{}).value);
  else if(a==='addr'){const v=($('coAddr')||{}).value;if(!v||!window.claude||!window.claude.lan)return;b.disabled=true;window.claude.lan.addHost(v).then(r=>{b.disabled=false;toast(r.ok?(r.games?`Found ${r.games} game${r.games>1?'s':''} there. Pick it from the list.`:'That computer is running Dementia Zombies but isn\\'t hosting a game yet.'):'Couldn\\'t reach that address. Check it, and that their firewall lets Dementia Zombies through.',r.ok?'haze':'bad');});}""")
rep("""$('ovCoop').addEventListener('keydown',e=>{if(e.target.id==='coCode'&&e.key==='Enter'){e.preventDefault();joinGame(e.target.value);}});""",
    """$('ovCoop').addEventListener('keydown',e=>{if(e.target.id==='coCode'&&e.key==='Enter'){e.preventDefault();joinGame(e.target.value);}if(e.target.id==='coAddr'&&e.key==='Enter'){e.preventDefault();const f=document.querySelector('[data-co="addr"]');if(f)f.click();}});""")

# saves live on this computer
rep("""'Slots are kept on your Claude account, so they follow you to other devices, with a copy in this browser.'""",
    """'Slots are saved on this computer (in AppData\\\\Roaming\\\\Dementia Zombies), with a copy in the game window. Exports go to Documents\\\\Dementia Zombies.'""")
rep("""$('svNote').textContent=`Exported ${fname}.`;""", """$('svNote').textContent=`Exported to ${window.__dzLastExport||fname}.`;""")
s += '\n</html>\n'

os.makedirs(os.path.join(ROOT, 'web'), exist_ok=True)
open(os.path.join(ROOT, 'web', 'game.html'), 'w', encoding='utf-8').write(s)
print('patched game.html', len(s))

# Windows resources: icon, version info, manifest
v = VERSION.replace('.', ',') + ',0'
manifest = '''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="connectrader.DementiaZombies" version="''' + VERSION + '''.0"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security></trustInfo>
  <application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings><dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true</dpiAware></windowsSettings></application>
</assembly>'''
open(os.path.join(ROOT, 'app.manifest'), 'w').write(manifest)
rc = f'''#include <winver.h>
1 ICON "icon.ico"
1 24 "app.manifest"
1 VERSIONINFO
FILEVERSION {v}
PRODUCTVERSION {v}
FILEOS 0x40004
FILETYPE 0x1
BEGIN
  BLOCK "StringFileInfo"
  BEGIN
    BLOCK "040904b0"
    BEGIN
      VALUE "CompanyName", "connectrader, LLC"
      VALUE "FileDescription", "Dementia Zombies"
      VALUE "FileVersion", "{VERSION}"
      VALUE "InternalName", "DementiaZombies"
      VALUE "OriginalFilename", "Dementia Zombies.exe"
      VALUE "ProductName", "Dementia Zombies"
      VALUE "ProductVersion", "{VERSION}"
    END
  END
  BLOCK "VarFileInfo"
  BEGIN
    VALUE "Translation", 0x409, 1200
  END
END
'''
rc = rc.replace('#include <winver.h>\n', '')
open(os.path.join(ROOT, 'app.rc'), 'w').write(rc)
WINDRES = next((t for t in ('llvm-windres-18', 'llvm-windres', 'x86_64-w64-mingw32-windres') if shutil.which(t)), None)
assert WINDRES, 'needs llvm-windres (apt install llvm-18)'
subprocess.check_call([WINDRES, '--target=pe-x86-64', '-O', 'coff', '-i', 'app.rc', '-o', 'rsrc_windows_amd64.syso'], cwd=ROOT)

os.makedirs(OUT, exist_ok=True)
env = dict(os.environ, GOOS='windows', GOARCH='amd64', CGO_ENABLED='0')
exe = os.path.join(OUT, 'Dementia Zombies.exe')
ld = f'-X main.version={VERSION}'
subprocess.check_call(['go', 'build', '-trimpath', '-ldflags', f'-s -w -H windowsgui {ld}', '-o', exe, '.'], cwd=ROOT, env=env)
# a Linux build of the same server, used for testing
subprocess.check_call(['go', 'build', '-ldflags', ld, '-o', os.path.join(OUT, 'dz-linux'), '.'], cwd=ROOT, env=dict(os.environ, CGO_ENABLED='0'))
print('built', exe, os.path.getsize(exe))

# packages, named with the version
readme = open(os.path.join(ROOT, 'README.txt'), encoding='utf-8').read()
readme = re.sub(r'^DEMENTIA ZOMBIES.*\n=+\n', f'DEMENTIA ZOMBIES v{VERSION} (Windows)\n' + '=' * len(f'DEMENTIA ZOMBIES v{VERSION} (Windows)') + '\n', readme, count=1)
for old in os.listdir(OUT):
    if old.endswith('.zip'):
        os.remove(os.path.join(OUT, old))
win = os.path.join(OUT, f'Dementia Zombies v{VERSION} (Windows).zip')
with zipfile.ZipFile(win, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as z:
    z.write(exe, f'Dementia Zombies v{VERSION}/Dementia Zombies.exe')
    z.writestr(f'Dementia Zombies v{VERSION}/README.txt', readme)
src = os.path.join(OUT, f'dementia-zombies-v{VERSION}-source.zip')
with zipfile.ZipFile(src, 'w', zipfile.ZIP_DEFLATED) as z:
    for fn in ['main.go', 'hub.go', 'ws.go', 'discover.go', 'update.go', 'release.py', 'platform_windows.go', 'platform_other.go', 'shim.js', 'build.py', 'icon.py', 'icon.ico', 'favicon.png', 'go.mod', 'README.txt']:
        z.write(os.path.join(ROOT, fn), f'dementia-zombies-v{VERSION}-source/{fn}')
    z.write(SRC, f'dementia-zombies-v{VERSION}-source/dementia-zombies.html')
print('packaged', os.path.basename(win), 'and', os.path.basename(src))

# release files for GitHub: the self-updater looks for *-windows-x64.exe and SHA256SUMS.txt
import hashlib
rel = os.path.join(OUT, 'release')
shutil.rmtree(rel, ignore_errors=True)
os.makedirs(rel)
rexe = os.path.join(rel, f'DementiaZombies-v{VERSION}-windows-x64.exe')
rzip = os.path.join(rel, f'Dementia-Zombies-v{VERSION}-Windows.zip')
shutil.copy(exe, rexe)
shutil.copy(win, rzip)
with open(os.path.join(rel, 'SHA256SUMS.txt'), 'w') as f:
    for p in (rexe, rzip):
        f.write(hashlib.sha256(open(p, 'rb').read()).hexdigest() + '  ' + os.path.basename(p) + '\n')
print('release files in', rel)
