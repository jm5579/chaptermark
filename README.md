# ChapterMark

A portable Windows app that remembers where you stopped reading every manga, manhwa and manhua — and tells you when new chapters have been translated.

- **Portable:** one `ChapterMark.exe` you double-click. No installer. Your list lives in a `data/` folder next to the exe, so copying the folder to another PC brings everything with it.
- **Finds any series by any name**, including alternate and native-script titles, using the free [MangaUpdates](https://www.mangaupdates.com) API (plus [MangaDex](https://mangadex.org) and [AniList](https://anilist.co) for covers, links and extra ratings).
- **Shows the latest translated chapter** for every series you're reading, sorted by your own rating.

> Status: under construction. See [docs/design.md](docs/design.md) for the full design.

## Running it

1. Download/copy the `ChapterMark` folder anywhere you can write to (Desktop, USB stick — not inside a zip, not Program Files).
2. Double-click `ChapterMark.exe`.
3. If you downloaded it from the internet, Windows may show **"Windows protected your PC"** the first time, because the exe isn't code-signed. Click **More info → Run anyway**.

Requires Windows 10 or 11 (uses the Microsoft Edge WebView2 runtime that ships with Windows).

## Where your data lives

```
ChapterMark/
  ChapterMark.exe
  data/
    library.db   your list (SQLite)
    covers/      cached cover images
    backups/     one copy per day you open the app (last 10 kept)
```

Use **Settings → Export** to save your whole list to a single file you can keep anywhere.

## Building from source

Requirements: [Go](https://go.dev/dl/) 1.23+, [Node.js](https://nodejs.org) 20+, and the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Then:

```bash
wails build
```

The exe is written to `build/bin/ChapterMark.exe`. For live development with hot reload, run `wails dev`.

Run the tests:

```bash
go test ./...
```

## License

MIT — see [LICENSE](LICENSE).
