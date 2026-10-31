# Changelog

## v1.9.0

**Polish**
- Menus, toasts and buttons animate in instead of snapping. The title flickers.
- Taking a hit leaves a pale chip on the health bar that drains after it, and the screen edges flush red.
- Cash counts up, and your kills, cash, level and skill points pop when they change.
- Level-ups ring out from you with a LEVEL banner and how many skill points you have to spend.
- Weapons in good condition and high-tier gear on the ground throw up a light beam in their quality colour, so you can spot them from across the screen.
- Cash, scrap, ammo, meds and other small pickups drift to you when you're close.
- Footstep dust, and the camera leans a little toward where you aim (further while sneaking).
- Death fades the screen out.

**Settings**
- A new Settings screen (Pause, the title screen or the quick menu): sound and music volume, screen shake (off, low, full), damage numbers, flashes, reduce motion, camera lean, compact HUD and a frame-rate counter. Saved on your device.
- Every keyboard key can be moved. Pick an action, press the new key. Taking a key that's in use swaps the two. Reset puts them all back.
- The controller screen is linked from Settings.

**Fixes and comfort**
- A gentler start: a brand-new character takes reduced damage for the first few minutes while learning the ropes. Off in hardcore.
- The status cards no longer overlap on small screens: they go compact on their own, showing only the essentials. Long text in them is cut off cleanly instead of spilling.
- The threat line is shorter.
- Special characters (dashes, symbols) display properly everywhere.
- A rare explosion with a missing size could break a frame. Fixed.

## v1.8.1

**Dementia can't run away from your level anymore**
- Fixed: unmeds multiplied your dementia (Fog pill +25%, Oblivion ampoule ×2, Riptide ×1.5, Void serum ×3), so a handful of pills could push it into the hundreds of thousands. The dead became tens of thousands of levels tougher than you and nothing could be killed. Pills now add a fixed amount scaled to your level, and stacked speed-ups cap at 6×.
- Dementia still makes the dead meaner and the loot better, but the further it runs ahead of your own level, the less each point adds, and it builds more slowly once it's well ahead.
- Dementia's experience bonus has a soft cap.
- Saves already caught in this are repaired once on load: dementia comes back within reach, runaway cash is trimmed, and items found during it are brought down to a reachable level.

**The game closing on you**
- The game window could close outright mid-run (Ctrl+W, or the window's X). It now asks before leaving a run, and Ctrl+W is blocked during play.
- Autosave every minute while you play, so a closed window costs a minute at most.
- A bug in one frame no longer freezes the game: it's logged, the game keeps going, and the log can be copied from Help.

## v1.8.0

**The long game: about 80 hours to 100%**
- Leveling is much slower and paced so a steady player reaches level 240 in about 80 hours. Experience per kill also eases when big crowds are coming at you.
- Level 240 is the main climb. After that, every level of experience is a Forgotten level and a point on an endless board: Might, Vigor, Focus, Swiftness, Fortune, Greed, Shadow and Reach. Spend them on the Skills screen.
- New Completion page (Pause or the quick menu): level 240, every difficulty, a dungeon cleared on each difficulty, every kind of dungeon, every stealth run (and each done unseen), every dimension, every named horror, the full bestiary and every class to level 40.

**Difficulty**
- Twenty tiers, from Normal through Hard, Expert, Master and Torment I to XVI, unlocking from level 1 to 235. Pick one in the new Difficulty tab at home and drop back any time.
- Higher tiers: tougher dead (1.2× health and 1.12× damage per tier), loot two item levels higher per tier, +25% cash and +6% experience per tier, and more elites.
- Each tier has a fixed range of weapon conditions and gear tiers, from Broken to Pristine on Normal up to Eldritch to Absolute on Torment XVI. Within a tier, drops lean toward the top of the range as you level.

**Stealth**
- Sneak: hold Shift, click the left stick, or use the quick menu. You move slower and quieter, and the dead notice you much later.
- Sneak attacks: anything that hasn't noticed you takes a much harder hit, harder still while sneaking and up close with a melee weapon.
- New gear stat Shadows (the dead notice you later), new weapon effect Assassin's (bigger sneak attacks) and the Forgotten Shadow stat. Quiet gear and the Scout class help too.
- Controller: the class skill moves to the right bumper and the left stick click is sneak. The left trigger swings.

**Stealth dungeons** (from level 8, mixed in with the other ways down)
- Heist: reach the vault at the far end. Guards patrol with visible vision cones and a detection meter; being seen sounds an alarm that wakes every one of them. An unseen heist doubles the vault.
- Ghost run: reach the far hatch without ever being seen. Spotted means thrown out.
- Assassination: kill the crowned marks. Guards that see a body raise the alarm; every alarm cuts the pay, and a clean job pays half again.

**Other**
- Dying no longer ends the run. You drop 10% of your cash and wake up in the house after a few seconds. For one life, start a Hardcore run from the title screen.
- Hordes start at level 8 and begin small.
- Dungeons keep getting deeper every 40 levels up to level 200.

## v1.7.0

**Power, crowds and readable numbers**
- Numbers grow far slower. Damage, health, armor, cash and prices now grow about 5% a level instead of 10 to 13%. Damage is in the thousands for the first stretch and around a million after an hour or two, instead of trillions.
- You get stronger and stay stronger. Single zombies keep dying faster as you grow; the dead no longer soak up your upgrades.
- The dead answer your power with numbers and nastiness instead: bigger crowds, faster spawns, more elites and more traits on them.
- Hordes: every minute or so a pack comes at you from one direction, with a warning. Packs grow from 5 to about 40 as you get stronger.
- Crowd effects scale with condition: Piercing (+1 per 4 condition levels), Twin (+1 per 6), Arcing (+1 jump per 4), Ricocheting (+1 bounce per 6), Splitting (+1 shard per 5) and Explosive (bigger blasts). They also roll more often.
- Bigger crowds mean more kills, so each kill gives a little less experience when the pressure is up.
- Saves from older versions are converted to the new numbers the first time they load.

## v1.6.0

**Dungeons**
- From level 6, a way down opens nearby every few minutes: the Root Cellar, the Storm Drains, St. Agnes Basement or the Fallout Shelter.
- Each is a fixed map of rooms and corridors with a fixed number of the dead. Nothing respawns and nothing wanders in, and a counter shows how many are left.
- A named horror waits in the last room. Clear everything and a chest opens with a pile of weapons and gear, cash, parts and experience.
- Dark past your light, with each kind dressed differently. Climb back out by the ladder any time.
- Deeper dungeons every 25 levels: more rooms, more of the dead, better loot.

**Dimensions**
- Every dimension has its own look: forest trails and log cabins in the Deep Woods, rows of graves and mausoleums in the Boneyard, striped market stalls and coin sacks in the Goblin Market, metal plating and crystal spires in the Grey Zone, red-lit brick and coffins in Nightfall Manor, and lava cracks and brimstone in the Pit.
- Each dimension is the only place certain effects drop. All lifesteal (Vampiric weapons, the Leech on gear and the Blood element) comes from Nightfall Manor only. Hellfire and Explosive come from the Pit, Seeking, Phantom and Gravitic from the Grey Zone, Unbroken and Executing from the Boneyard, Fleet and Deadshot from the Deep Woods, and Lucky and Scavenging from the Goblin Market. Loot found there often carries one.
- Things you already own keep what they have.

**Soft caps**
- Gear bonuses (damage, attack speed, crit, health, cash, experience, luck, lifesteal, thorns, range and more) count in full up to a point, then each extra bit is worth less.
- Counter treatments (Vitamins, Whetstone kit, Coin purse) and the endless skills follow the same curve, and show what the next rank actually adds.
- Levels a weapon or piece of gear gains after it drops (tuning, bond catch-up, heirloom scaling) count for less the further it goes. Better condition holds its value longer. Keeping a favourite still works for a good while, but new finds catch up.

## v1.5.0

**Balance overhaul: a run that lasts**
- Fixed the runaway that made the game pointless after about ten minutes. Class bonuses to experience, luck, cash and parts grew without limit, which fed more experience, which grew them further. Levels went into the millions while you couldn't be hurt and nothing in the shop mattered.
- Class passives now grow until class level 100, where the class is mastered.
- The endless skills (Trigger Discipline, Butcher's Rhythm, Stubborn, Embrace the Fog) give a flat bonus per point instead of compounding.
- Levels come steadily: quick at first, about a minute each by the mid-game, slower after that. The "made it home" bonus no longer gets multiplied twice.
- Elites are a little rarer and give a little less experience; hard events double them instead of tripling.
- The dead size you up. Every second the game checks how quickly you drop a plain zombie and how many hits you could take, and new zombies close most of the gap to a fair fight over about half a minute. Upgrades still feel big, but no amount of stacking makes you untouchable.
- Zombies scale harder with level to keep pace with what a long run piles on.

**Something new for hours**
- New kinds of dead now arrive from level 3 to level 75, instead of all of them by 26.
- Elite traits unlock one at a time up to level 74. Elites can carry two traits from level 60 and three from level 120.
- Named horrors join the roster as you go: four when hunts start at level 22, then one more every six levels.
- Rifts open from level 20, with a new dimension at 32, 44, 56, 70 and 85.
- Events are spread from level 1 to level 40. Loot goblins appear from level 12 (relic goblins at 25, golden at 40), crazed survivors from level 8.
- A message tells you whenever something new is out there.

## v1.4.0

**Menus on a controller**
- Every menu is now a vertical list you roll through with up and down. The highlighted item stays centred as you go, and the list wraps around at the ends.
- Something is always highlighted: menus open on their first item, come back to where you left off, and keep your place after you buy, sell or equip.
- Up and down follow what's on screen: straight down a skill tree, straight down the shop shelf. Left and right move across a row.
- The house tabs are on the bumpers (LB and RB on screen), and left or right at the edge of a list flips the tab. In skill trees the bumpers jump between trees.
- Dropdown settings change with left and right (or A), no pop-up lists.
- A controls strip along the bottom of every menu shows what each button does on your controller.
- The Controller settings screen can now be navigated like everything else.
- Long screens like Help scroll with the D-pad.
- Mouse and keyboard are unchanged; the controller layout switches on when you pick up the controller and off when you move the mouse. Arrow keys now move through menus too.

## v1.3.0

**Controllers**
- Start always opens the pause screen: from play, from inside the house, and from any menu. Press it again to resume.
- New quick menu on Select (View, Share or Minus): every screen in the game, plus actions that fit where you're standing, like opening the house, taking meds, throwing, your class skill, markers and chat in co-op. Nothing needs a button of its own anymore. Bag moved into the quick menu and can still be bound to a button.
- Xbox Series controllers connected over Bluetooth showed PlayStation button names. Fixed.

## v1.2.4

**Updates**
- "Check for updates" is now on the title screen, not just the pause menu.
- The game checks for a new version as soon as it opens and shows it on the title screen with an "Install and restart" button.
- A clearer message when GitHub has no public release to check.

## v1.2.3

**Controllers**
- Fixed: on some controllers, holding a stick all the way in one direction for a couple of seconds could scramble the layout. Fire jumped between triggers, the sticks swapped and aiming stopped turning all the way around. Layouts now stay exactly as detected or as you set them.

## v1.2.2

**Controllers**
- The aim line and reticle are now off by default. Turn them back on with "Aim reticle" under Pause → Controller.

## v1.2.1

**Controllers**
- Pushing the right stick no longer fires on its own. Aim freely and fire with the right trigger.
- New "Right stick fires" option under Pause → Controller for twin-stick style shooting.
- Controllers whose triggers report the wrong resting position until first pressed are now corrected automatically, so they can't look like a held trigger.

## v1.2.0

**Controllers**
- Works with any controller: Xbox, PlayStation, Switch Pro, 8BitDo and generic USB pads. Non-standard pads are set up automatically, including triggers that report as axes, d-pads that report as a single hat, and right sticks on unusual axes.
- New Pause → Controller screen: live input test, rebind every action (saved per controller), aim assist (off, low, normal, strong), stick dead zone and vibration.
- True 360° aiming: smooth analog aim in any direction, a sight line and reticle on screen, and the reticle distance follows how far you push the stick. Grenades, targeted class skills and map markers land on the reticle.
- On-screen prompts use your controller's own button names (A, ✕ or B).
- Vibration on hits, explosions, the low-health heartbeat, level-ups, streaks and boss arrivals.
- Menus: the left stick navigates with repeat, the bumpers switch tabs at the house, the right stick scrolls.
- Whichever controller you touched last is in charge, and the game pauses if it disconnects mid-play.
- Aim assist now defaults to low.

## v1.1.0

**Playing together**
- Live ping for every player: next to each name in the party list, on the Co-op screen, and for every game in the network game list.
- Respawning: dying in a co-op world no longer ends your run. You drop 10% of your cash and wake up in the house 8 seconds later.
- Catch-up help for anyone below the leader's level: more experience, more damage, less damage taken and better loot, scaled to the gap.
- Map markers (Z) and text chat (Y).
- The host can kick players from the Co-op screen.
- A dropped connection gets 8 seconds to come back before the player's spot is given up, with a "Reconnecting" notice.

**Updates**
- The Windows app checks GitHub for new versions at launch and from the pause menu, and can install and restart itself.
- Version number on the title screen, the pause menu, in exported saves and in every package name.

**New in this version since the first Windows build**
- Dimensions: rifts to the Goblin Market, the Deep Woods, the Boneyard, the Grey Zone, Nightfall Manor and the Pit.
- Loot goblins (loot, money, weapon, armor, relic, scrap, golden) that run and portal away.
- Crazed survivors armed with real weapons.
- 10 new kinds of dead, 7 elite traits, 8 bosses, 8 world events, snow and ashfall, 6 pets, 6 unmeds, 8 places to clear, 9 outfits.
- Kill streaks, hit-stop, lasting gore, a low-health heartbeat, reactive music and full controller support.
- A faster first ten minutes, and a full balance pass.
- The safehouse no longer lands on places, camps, drops or bosses.
