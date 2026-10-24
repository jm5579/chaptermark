# ChapterMark

A portable Windows app that remembers where you stopped reading every manga, manhwa and manhua — and tells you when new chapters have been translated.

- **Portable:** one `ChapterMark.exe` you double-click. No installer. Your list lives in a `data/` folder next to the exe, so copying the folder to another PC brings everything with it.
- **Finds any series by any name**, including alternate and native-script titles, using the free [MangaUpdates](https://www.mangaupdates.com) API (plus [MangaDex](https://mangadex.org) and [AniList](https://anilist.co) for covers, links and extra ratings).
- **Shows the latest translated chapter** for every series you're reading, sorted by your own rating.

The full design is in [docs/design.md](docs/design.md).

## Running it

1. Download `ChapterMark-vX.Y.Z-windows.zip` from [Releases](https://github.com/jm5579/chaptermark/releases) and extract the `ChapterMark` folder anywhere you can write to (Desktop, Documents, a USB stick — not inside the zip, not Program Files).
2. Double-click `ChapterMark.exe`.
3. If you downloaded it from the internet, Windows may show **"Windows protected your PC"** the first time, because the exe isn't code-signed. Click **More info → Run anyway**.

Requires Windows 10 or 11 (uses the Microsoft Edge WebView2 runtime that ships with Windows). Nothing else needs to be installed.

## Using it

- **Add a series:** type any of its names in the search box — English, romanized (e.g. *Jeonjijeok Dokja Sijeom*), or native script (e.g. *전지적 독자 시점*). Results from MangaUpdates appear under *Add from MangaUpdates*; click **Add**, set the last chapter you read and your rating.
- **Pick the type:** the **Manhwa / Manga / Manhua** switch at the top shows one type at a time. Each button says how many series of that type have new chapters.
- **See what's new:** the *New chapters* row shows everything you're reading (or have on hold) that has chapters you haven't read yet, highest-rated first, e.g. `212 → 248 · +36`.
- **Update your progress:** click a series and use **+1**, or type the chapter. You can also set the status, your rating (1–10), notes, and a link to where you read it.
- **Latest chapter:** ChapterMark checks MangaUpdates' release list for the highest translated chapter (it shows which group released it). It checks automatically when you open the app (every 6 hours by default) and whenever you click **↻ Checked …**. If the sources are behind, set the latest chapter manually in the series panel.

## Where your data lives

```
ChapterMark/
  ChapterMark.exe
  data/
    library.db   your list (SQLite)
    covers/      cached cover images
    backups/     one copy per day you open the app (last 10 kept)
```

To move to another PC, copy the whole `ChapterMark` folder. Use **Settings → Export list…** to save your whole list to a single file you can keep anywhere, and **Import list…** to merge it back or restore it.

## Building from source

Requirements: [Go](https://go.dev/dl/) 1.23+, [Node.js](https://nodejs.org) 20+, and the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
```

Then:

```bash
wails build
```

The exe is written to `build/bin/ChapterMark.exe` (its data folder is `build/bin/data`). For live development with hot reload, run `wails dev`.

Run the tests:

```bash
go test ./...
```

```bash
cd frontend && npm test
```

## Releasing

Push a version tag and GitHub Actions builds the exe and publishes `ChapterMark-<tag>-windows.zip` as a release:

```bash
git tag v1.0.0 && git push origin v1.0.0
```

## Data sources

ChapterMark uses the free public APIs of [MangaUpdates](https://api.mangaupdates.com), [MangaDex](https://api.mangadex.org/docs/) and [AniList](https://docs.anilist.co) — no accounts or keys. It identifies itself with a `ChapterMark/1.0` User-Agent and spaces out its requests (about one per second to MangaUpdates).

## License

MIT — see [LICENSE](LICENSE).
