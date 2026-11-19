# Updater key rotation

The Console and Synnax Desktop updaters trust one public key, `pubkey` in
`console/src-tauri/tauri.conf.json`. The Core updater trusts each key in
`release.DefaultConfig.PublicKeys` (`core/pkg/release/release.go`). The secrets
`TAURI_PRIVATE_KEY` and `TAURI_KEY_PASSWORD` sign every Console, Desktop, Core, and
Driver release.

An installed app rejects an update that its own key cannot verify. So a new key must
first ship in a release that the old key signs. That release is the bridge.

## Rotate the key

Do the steps in order. Do not start a release that this list does not name until step 4
is done.

1. Add the new public key to the Core's `PublicKeys`, ahead of the old key. Merge it,
   and let it ship in a normal release.
2. Merge the change of `pubkey` in `tauri.conf.json` to `main` just before the next
   stable minor. Dispatch that minor with `prerelease` off. No release candidate ships
   from `main` between the merge and this release.
3. Cherry-pick the same change onto the newest `release/desktop-X.Y` branch. Release a
   stable Desktop patch from it with `prerelease` off. It must be the highest stable
   Desktop, so that every Desktop install updates to it.
4. After both releases publish, set `TAURI_PRIVATE_KEY` and `TAURI_KEY_PASSWORD` to the
   new key.
5. In a later release, remove the old key from the Core's `PublicKeys`.

## What breaks

- A release candidate that carries the new key, before step 4, strands the `next`
  channel. A stable release signed with the old key after a candidate that trusts only
  the new key is rejected.
- A release from a branch without the bridge, after step 4, cannot reach the installs of
  that branch. The older Console trains get no bridge, so their users reinstall from the
  download page to get updates. Desktop needs no bridge per train: every Desktop install
  updates to the highest stable Desktop on any train, so it reaches the bridge of
  step 3.
- A Console from before step 2 trusts only the old key, however it was installed. After
  step 4 it cannot update itself, so its user reinstalls from the download page.
- A Console offers the Console of an older Core's minor as a downgrade when it connects
  to that Core over TLS. Before step 4 the old key signs that downgrade, so a Console
  from step 2 or later rejects it, and the user installs it by hand. After step 4 a
  downgrade signed with the new key installs, but the Console it installs is one of the
  Consoles in the item above.
- A Core older than step 1 trusts only the old key. It must update to a Core from step 1
  or later before step 4, or be updated by hand.
