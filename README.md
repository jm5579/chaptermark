<div align="center">

<img src="build/appicon.png" width="112" alt="ChapterMark icon">

# ChapterMark

**Never lose your place in a manga, manhwa or manhua again.**

A portable Windows app that remembers the last chapter you read of every series — whichever site you read it on —<br>
and tells you when new chapters have been translated.

[![Latest release](https://img.shields.io/github/v/release/jm5579/chaptermark?style=flat-square&color=ff8a5c&label=release)](https://github.com/jm5579/chaptermark/releases/latest)
[![Tests](https://img.shields.io/github/actions/workflow/status/jm5579/chaptermark/ci.yml?branch=main&style=flat-square&label=tests)](https://github.com/jm5579/chaptermark/actions/workflows/ci.yml)
[![Windows 10 | 11](https://img.shields.io/badge/Windows-10%20%7C%2011-0078D6?style=flat-square&logo=windows&logoColor=white)](#-getting-started)
[![Portable](https://img.shields.io/badge/install-none%20·%20portable-3a3542?style=flat-square)](#-where-your-data-lives)
[![License: MIT](https://img.shields.io/github/license/jm5579/chaptermark?style=flat-square&color=3a3542)](LICENSE)

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails_v2-DF0000?style=flat-square)](https://wails.io)
[![React](https://img.shields.io/badge/React-20232A?style=flat-square&logo=react&logoColor=61DAFB)](https://react.dev)
[![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org)

### [⬇ Download for Windows](https://github.com/jm5579/chaptermark/releases/latest)

<br>

<img src="docs/screenshots/home.png" alt="ChapterMark home screen: new chapters sorted by your rating, with your list below" width="900">

</div>

<br>

## ✨ Features

<table>
  <tr>
    <td width="34%">🔖 <b>Remembers your place</b></td>
    <td>Last chapter read, your rating, status and notes for every series — in one place, whatever site you read on.</td>
  </tr>
  <tr>
    <td>🆕 <b>New chapters at a glance</b></td>
    <td>Finds the highest translated chapter on MangaUpdates (and which group released it). Series with new chapters float to the top, sorted by your rating: <code>276 → 311 · +35</code>.</td>
  </tr>
  <tr>
    <td>🔎 <b>Finds any series by any name</b></td>
    <td>English, romanized (<i>Jeonjijeok Dokja Sijeom</i>) or native script (<i>전지적 독자 시점</i>) — they all find the same series.</td>
  </tr>
  <tr>
    <td>🗂️ <b>Manhwa · Manga · Manhua</b></td>
    <td>One switch per type, each showing how many of its series have new chapters.</td>
  </tr>
  <tr>
    <td>⭐ <b>Your rating, next to everyone's</b></td>
    <td>Rate 1–10 and see the MangaUpdates and AniList scores alongside.</td>
  </tr>
  <tr>
    <td>💾 <b>Truly portable</b></td>
    <td>One <code>.exe</code>, no installer, no account, no traces left on the PC. Everything lives in a <code>data</code> folder next to the exe — copy the folder and keep going on another computer.</td>
  </tr>
  <tr>
    <td>🛟 <b>Safe with your list</b></td>
    <td>A backup every day you open it (last 10 kept), one-click Export/Import, and your list opens fine offline.</td>
  </tr>
</table>

## 📸 Screenshots

<table>
  <tr>
    <td width="50%"><img src="docs/screenshots/search.png" alt="Searching by a Korean romanized name"></td>
    <td width="50%"><img src="docs/screenshots/detail.png" alt="The series panel"></td>
  </tr>
  <tr>
    <td align="center"><b>Search by any name</b><br><sub>Your list first, then MangaUpdates — showing which alternate name matched</sub></td>
    <td align="center"><b>Update your progress</b><br><sub>+1 per chapter, rating, status, notes and where you read it</sub></td>
  </tr>
</table>

## 🚀 Getting started

1. Download **`ChapterMark-vX.Y.Z-windows.zip`** from [Releases](https://github.com/jm5579/chaptermark/releases/latest).
2. Extract the `ChapterMark` folder anywhere you can write to — Desktop, Documents, a USB stick.
3. Double-click **`ChapterMark.exe`**. That's it — nothing to install.

> [!NOTE]
> The first time you run a downloaded copy, Windows may show **"Windows protected your PC"** because the exe isn't code-signed. Click **More info → Run anyway**.

Requires Windows 10 or 11 (it uses the Microsoft Edge WebView2 runtime that ships with Windows).

## 📖 Using it

- **Add a series** — type any of its names in the search box. Results from MangaUpdates appear under *Add from MangaUpdates*; click **Add** and set the last chapter you read and your rating.
- **Pick the type** — the **Manhwa / Manga / Manhua** switch shows one type at a time.
- **See what's new** — the *New chapters* row lists everything you're reading (or have on hold) with unread chapters, highest-rated first.
- **Update your progress** — click a series and press **+1**, or type the chapter. Set status, rating, notes and a link to where you read it.
- **Stay current** — ChapterMark checks for new chapters when you open it and every few hours while it's open (every 6 hours by default — change it in ⚙ Settings), or whenever you click **↻ Checked …**. If the sources are behind, set the latest chapter manually in the series panel.

## 🗄️ Where your data lives

```
ChapterMark/
├── ChapterMark.exe
└── data/
    ├── library.db   ← your list (SQLite)
    ├── covers/      ← cached cover images, so it works offline
    ├── backups/     ← one copy per day you open the app (last 10 kept)
    └── webview/     ← the built-in browser's cache
```

> [!TIP]
> To move to another PC, copy the whole `ChapterMark` folder. **⚙ Settings → Export list…** saves your whole list to a single file you can keep anywhere; **Import list…** merges it back or restores it.

## 🔌 Data sources

ChapterMark uses the free public APIs of [MangaUpdates](https://api.mangaupdates.com), [MangaDex](https://api.mangadex.org/docs/) and [AniList](https://docs.anilist.co) — no accounts or keys. It identifies itself with a `ChapterMark/1.0` User-Agent and spaces out its requests (about one per second to MangaUpdates).

<sub>Cover art and titles belong to their respective creators and publishers. ChapterMark is a personal tracker and isn't affiliated with MangaUpdates, MangaDex or AniList.</sub>

## 🛠️ Building from source

<details>
<summary><b>Build, test and release</b></summary>

<br>

Requirements: [Go](https://go.dev/dl/) 1.23+, [Node.js](https://nodejs.org) 20+, and the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
```

Build the exe (written to `build/bin/ChapterMark.exe`, with its data in `build/bin/data`):

```bash
wails build
```

Live development with hot reload:

```bash
wails dev
```

Run the tests:

```bash
go test ./...
```

```bash
cd frontend && npm test
```

**Releasing:** push a version tag and GitHub Actions builds the exe and publishes `ChapterMark-<tag>-windows.zip` as a release:

```bash
git tag v2.1.0 && git push origin v2.1.0
```

</details>

## 🕰️ History

- **v2 — ChapterMark:** this portable Windows app (Go + Wails + React), a rewrite of the original tracker.
- **v1.0.0 — the original Python manga tracker.** Its code is kept in the repository history: `git checkout v1.0.0`.

## 📄 License

MIT — see [LICENSE](LICENSE).
