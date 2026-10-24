# ChapterMark — Design

*Date: 2026-10-23 · Status: approved by owner 2026-10-23*

## 1. Intent

The owner reads a lot of manga (Japanese), manhwa (Korean) and manhua (Chinese) across several different reading websites. Series run to hundreds of chapters. After a break of weeks or months they forget which chapter they stopped at, and they don't know whether new chapters have been translated since.

ChapterMark is a personal tracker that is the single source of truth for "where did I stop, and is there more to read now?" — independent of whichever site the reading happens on. It is **a tracker, not a reader**: it never displays chapters.

### What success looks like

The owner opens the app after a few weeks away and immediately sees which series have new translated chapters, ordered by how much they like each series, with their last-read chapter next to the latest available chapter (e.g. `212 → 248 · +36`).

## 2. Goals

1. Track, per series: last chapter read, personal rating, reading status, and when it was last read.
2. Find any series by typing a name — including alternate names (English, romanized Korean/Japanese/Chinese, native script). Searching "Jeonjijeok Dokja Sijeom", "ORV" or "Omniscient Reader's Viewpoint" all find the same series.
3. Show the **latest translated (English) chapter** for each tracked series and how many chapters are new since the owner's last-read chapter.
4. Show general/public ratings from the data sources alongside the owner's own rating.
5. Be **portable**: a folder containing an exe that runs by double-click, no installer. Copying the folder to another Windows PC carries all saved data with it.
6. A good-looking, modern UI.
7. Use only free services; no accounts or API keys required.

## 3. Non-goals (not in v1)

- Reading chapters or linking into specific chapter pages on reading sites.
- Automatic progress tracking (the owner updates the chapter manually every time they read — they said this is easier).
- macOS / Linux / mobile versions.
- Syncing between computers over the internet (portability is by copying the folder).
- Importing existing lists from MAL/AniList/MangaUpdates accounts.
- Notifications while the app is closed / running in the background.
- Light novels / web novels.
- Series that aren't on MangaUpdates (hand-entered series). *[my call: MangaUpdates covers nearly everything; revisit if the owner hits a gap]*
- Code-signing the exe (costs money; see the SmartScreen note in §9).

## 4. Anti-goals (would make it a failure even if it "works")

- Requiring installation, admin rights, or a sign-up anywhere.
- Storing data outside the app folder (e.g. `%APPDATA%`), which would break "copy the folder to another PC".
- Losing the owner's list (corruption with no backup).
- The list being unusable offline — the owner's own data must always open, even without internet.
- Hammering the free APIs (getting rate-limited or blocked).

## 5. Decided by the owner

- **Portable Windows app**, double-click to run, no installer. Data travels with the folder.
- **Data from public databases via API**, MangaUpdates first; other sources welcome.
- **Owner updates progress manually.**
- **Home screen = layout "C"** (mockup approved):
  - Top bar: one search box + a "Checked N min ago ↻" button.
  - **Type switch: Manhwa / Manga / Manhua**, opens on **Manhwa**. Everything below shows only the selected type.
  - **New chapters shelf**: cover cards for series with new chapters, sorted by the owner's rating. Each card: cover, `last read → latest`, `+N` badge, owner rating.
  - **List below** with tabs: Reading · Plan to read · Completed · On hold · Dropped. Each row: cover thumbnail, title, alternate names, progress bar (`212 / 248`), `+N` new badge, owner rating, MangaUpdates rating, "read N ago".
- **New chapters first, then by the owner's rating.**
- A new, separate project and a new GitHub repo — unrelated to any earlier work.
- **Name: ChapterMark.**
- **Stack: Go + Wails** (after weighing it against Python + pywebview; see §6.1). Go may be installed on this PC to build it.
- **Repo: public**, under the owner's GitHub account `jm5579`.

The approved mockup is reproduced in the appendix (§13).

## 6. Technical approach (builder's recommendations — marked *[my call]*)

### 6.1 Stack *[recommended by builder, chosen by owner]*

- **Wails (Go backend) + React + TypeScript (Vite) frontend.** Produces one native `ChapterMark.exe` (~15 MB) that renders its UI with Microsoft Edge WebView2 (preinstalled on Windows 10/11; present on the owner's PC, v155).
- **SQLite** via a pure-Go driver (`modernc.org/sqlite`) — no CGO, no DLLs next to the exe.
- Rejected: Electron (~250 MB folder, slower start), Tauri (needs ~6 GB Rust + MSVC toolchain), Python + pywebview (PyInstaller exes are large, unpack to temp on every launch, and frequently trip antivirus false positives), .NET (large self-contained exe or needs runtime on target PC).
- Build machine needs Go installed (one-time, ~250 MB). Node 24, git, and `gh` (logged in as `jm5579`) are already present.

### 6.2 Portable layout

```
ChapterMark/
  ChapterMark.exe
  data/                 ← created on first run, next to the exe
    library.db          ← SQLite database (all owner data + cached metadata)
    covers/             ← downloaded cover images, so the list renders offline
    backups/            ← library-YYYY-MM-DD.db, one per day the app is opened; keep last 10
```

- The data directory is resolved **relative to the exe's own location**, never the working directory or `%APPDATA%`.
- If the folder is read-only (e.g. inside a zip or Program Files), show a clear message telling the owner to extract/move the folder.
- **Export / Import**: a button writes the whole library to a single JSON file the owner can keep anywhere; import restores from it (replace or merge — builder decides, must confirm before overwriting).

### 6.3 Data sources (verified against live APIs on 2026-10-23)

| Source | Used for | Notes from live testing |
|---|---|---|
| **MangaUpdates API v1** (`api.mangaupdates.com/v1`) — primary | Search (matches alternate names), series details, **type** (`Manga`/`Manhwa`/`Manhua`/…), all associated names, Bayesian rating (/10), release list with group names | Searching `jeonjijeok dokja sijeom` returned *Omniscient Reader* (Manhwa, 8.82, 8+ alt names) as the top hit. Search returns 25 results per page. |
| **MangaDex API** (`api.mangadex.org`) | Cross-linking to AniList / MAL / MangaUpdates IDs, extra alt titles, cover art fallback | `attributes.links.mu` is the **MangaUpdates series_id in base-36** (`n50wl4o` = `50369844984`, verified). Also has `links.al`, `links.mal`, `links.engtl` (official English source). **Not used for chapter counts**: licensed series are mostly removed (ORV has only 4 English chapters there). |
| **AniList GraphQL** (`graphql.anilist.co`) | Second public rating (`averageScore` /100, shown as /10), cover fallback | ORV `averageScore` = 86. `chapters` is null for ongoing series, so not useful for latest chapter. |

All three are free and need no API key. Every client must send an identifying `User-Agent` (e.g. `ChapterMark/1.0 (personal tracker)`), respect rate limits, and back off on HTTP 429/5xx.

### 6.4 Latest translated chapter — the key logic

- **Latest chapter = the highest chapter number found in the MangaUpdates release list for that series**, *not* the series' `latest_chapter` field.
  - Evidence: for ORV, `latest_chapter` = 308 (the official LINE Webtoon release), while the release list contains chapter 311 from a scanlation group (Flame Comics). The owner reads on many sites, so the highest translated chapter is what matters.
- Store alongside it the **group name** and **release date** of that chapter; show them in the detail panel (e.g. "311 · Flame Comics · 19 May 2026").
- Chapter strings are messy: `"311"`, `"10.5"`, `"1-5"`, `"c.45"`, `"v.3 c.12"`, `"Season 2 Ep. 4"`. Parse to the highest numeric chapter in the string; ignore entries with no parseable chapter. Unit-test the parser against a table of real examples.
- **New count** = `floor(latest) − floor(my_chapter)`, minimum 0. Show `+N` only when > 0.
- Known limitation (accepted): if a series restarts numbering per season, or a source lags, the number can be off. The owner can always see the group/date in the detail panel. *[my call]* Manual override: the owner may type a "latest chapter" by hand for a series; a manual value is kept until a fetched value exceeds it.

### 6.5 Checking for new chapters

- On app start, if the last full check is older than **6 hours** *[my call]*, check automatically in the background; the "Checked … ↻" button forces a check any time.
- Check series in status **Reading, On hold, Plan to read**; skip Completed and Dropped *[my call]*.
- Order: highest owner rating first, so the most important series update first.
- Throttle MangaUpdates to ~**1 request/second** *[my call]*; results stream into the UI as they arrive; the button shows progress (`Checking 34/120…`). A few hundred series taking a few minutes is acceptable.
- Builder may optimise (e.g. MangaUpdates' recent-releases-by-day endpoint for frequent checks, per-series lookup only for stale ones) as long as results are equivalent.
- Offline or API failure: keep last known values, show "Couldn't check — offline" and the time of the last successful check. Never blank out stored data.
- Re-download cover and refresh ratings/alt names at most once every **7 days** per series *[my call]*.

### 6.6 Search and add

- **One search box** in the top bar.
  - Section 1, **"In your list"**: instant, offline match against title **and every alternate name**, case/diacritic/punctuation-insensitive (so "omniscient readers viewpoint" matches "Omniscient Reader's Viewpoint"). Native scripts (한글, 漢字, かな) match as typed.
  - Section 2, **"Add from MangaUpdates"**: online results (debounced ~400 ms), each with cover, title, matched alt name, type, year, MU rating and an **Add** button. Already-tracked series show "In your list" instead of Add.
  - Show Manga / Manhwa / Manhua by default; other MU types (Novel, Doujinshi, OEL, Artbook…) hidden behind a "show other types" toggle *[my call]*.
- **Adding**: choose a status (default Reading) and optionally a current chapter (default 0) and rating. The app then fetches full details + releases, downloads the cover, and tries to link MangaDex/AniList (search MangaDex by title, accept the result whose `links.mu` matches the MU id; take `links.al` for AniList). If linking fails, the series still works on MangaUpdates data alone.
- A series' **type comes from MangaUpdates** and decides which switch tab it appears under. Owner can override the type per series *[my call]*.

### 6.7 Screens

1. **Home** — as in §5 / appendix. Sort within the selected type and tab: series with new chapters first, then owner rating (high → low; unrated last), then most recently read.
   - *[my call]* The **New chapters shelf** and the **"N new" counts** on the type switch include only series in **Reading** and **On hold**. A *Plan to read* series has no progress yet, so its row shows the total available (`311 ch`) instead of a `+311` badge — otherwise every unstarted series would flood the shelf.
2. **Series detail panel** (slides in when a cover or row is clicked):
   - Cover, title, all alternate names, type, year, publication status text from MU.
   - **My chapter** with `−1` / `+1` buttons and a number field (decimals allowed, e.g. 10.5). Changing it updates "last read" to now.
   - Latest chapter + group + date; manual latest override.
   - My rating, my status, notes (free text), and an optional **"where I read it" URL** *[my call: handy because the owner reads on multiple sites]*.
   - Public ratings: MangaUpdates /10, AniList /10 (converted from /100), with vote counts where available.
   - Link to open the MangaUpdates page in the default browser. Remove-from-list button (with confirm).
3. **Settings** (small): auto-check interval, number of backups to keep, Export / Import, "open data folder".

- **Owner rating scale**: 1–10, whole numbers *[my call]* — matches MangaUpdates' /10 so they sit side by side.
- **Visual style**: dark theme as in the mockup (near-black background, warm orange accent, star-yellow for the owner's rating). Real cover art replaces the placeholder gradients. Should feel like a polished media app, not a spreadsheet.
- Window: resizable, remembers size/position; minimum ~1000×680. Layout adapts the shelf card count to width.

### 6.8 Data model (SQLite, sketch — builder may refine)

- **series** — local id; `mu_id` (unique), `md_id`, `al_id`, `mal_id`; `title`; `type` (`manhwa`/`manga`/`manhua`/`other`) + `type_override`; `year`; `pub_status` (MU text); `completed` (bool); `cover_file`; `mu_rating`, `mu_votes`, `al_rating`; `latest_chapter` (real), `latest_raw` (text), `latest_group`, `latest_date`, `latest_manual` (real, nullable); `details_refreshed_at`, `chapters_checked_at`.
- **alt_titles** — `series_id`, `title`, `normalized` (for search). Indexed; consider FTS5.
- **entry** (the owner's tracking; 1:1 with series in v1) — `status` (`reading`/`plan`/`completed`/`on_hold`/`dropped`), `my_chapter` (real), `my_rating` (int 1–10, nullable), `notes`, `read_url`, `last_read_at`, `added_at`, `updated_at`.
- **meta** — key/value: schema version, `last_full_check_at`, settings, window state.
- Schema migrations versioned from day one (owner's data must survive upgrades). All timestamps stored as UTC ISO-8601.

### 6.9 Reliability

- SQLite in WAL mode; daily backup copy on open (keep 10); write the backup *before* running any schema migration.
- All network work happens in the Go backend (no CORS issues, one place for throttling/back-off/timeouts ≈ 15 s).
- UI never blocks on the network; stale data is shown with its age.

## 7. Project, repo, build

- Project folder: `C:\Users\Joe\Desktop\Projects\chaptermark` (alongside the owner's other projects, fully separate from them).
- New **public** GitHub repo `jm5579/chaptermark` (owner's choice). This document is committed as `docs/design.md`.
- Because the repo is public: `.gitignore` must exclude `data/`, `build/bin/`, any `*.db`, and exported JSON files, so the owner's list never reaches GitHub. No secrets exist (no API keys needed). *[my call]* MIT license.
- `wails build` produces `build/bin/ChapterMark.exe`. A release step zips `ChapterMark/ChapterMark.exe` (data folder created on first run).
- *[my call]* GitHub Actions workflow on a `windows-latest` runner: run tests on every push; on a version tag, build the exe and attach `ChapterMark-vX.Y.Z-windows.zip` to a GitHub Release.
- README: what it is, how to run, where data lives, how to build.

## 8. Testing expectations

- Go unit tests: chapter-string parser (table of real examples), "new count" and sort order, title normalisation/alt-name matching, base-36 MU-id conversion, migrations, data-dir resolution.
- API clients tested against **recorded JSON fixtures** (no live calls in CI), plus one opt-in live smoke test.
- Frontend: component tests for the home screen's grouping/sorting/type switch; manual check of the built exe on a clean folder (first run, add series, restart offline, copy folder elsewhere and reopen).

## 9. Constraints

- Windows 10/11 only; relies on WebView2 (preinstalled). If missing, Wails' standard prompt is acceptable.
- Free APIs only; may change or rate-limit — the app must degrade gracefully, never lose owner data.
- English translations are what "latest chapter" means.
- The exe is unsigned. Copied via USB/network it just runs; **downloaded** (e.g. from a GitHub Release zip) Windows SmartScreen will show "Windows protected your PC" the first time → *More info → Run anyway*. The README must say this.

## 10. Left to the builder

App icon, component library / styling approach, exact visual spacing and typography, state management, Go package structure, migration tooling, exact throttle/back-off numbers within the politeness limits above, FTS vs LIKE for local search, merge-vs-replace semantics on import (with confirmation), and the precise chapter-parsing rules (given the examples and tests above).

## 11. Open questions

None. Stack, name and repo visibility were decided by the owner on 2026-10-23 (§5).

## 12. Glossary

- **Manga / Manhwa / Manhua** — Japanese / Korean / Chinese comics.
- **Scanlation group** — fan translators who release chapters (e.g. Flame Comics); official publishers (e.g. LINE Webtoon) also appear in MangaUpdates' release list.
- **MU** — MangaUpdates. **MD** — MangaDex. **AL** — AniList.

## 13. Appendix — approved home screen (layout C)

```
┌──────────────────────────────────────────────────────────────────────────┐
│ CHAPTERMARK [ Search your list or add a new series…      ] [↻ Checked 2m]│
│                                                                          │
│ [ Manhwa ·5 new ] [ Manga ·2 new ] [ Manhua ·1 new ]   ← opens on Manhwa │
│                                                                          │
│ NEW CHAPTERS · 5   (sorted by your rating)                               │
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐                             │
│ │ +36  │ │ +33  │ │ +21  │ │  +9  │ │ +12  │   cover cards               │
│ │ ORV  │ │MtHua │ │ Nano │ │TBATE │ │SL Rag│                             │
│ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘                             │
│ 212→248★10 98→131★9 180→201★9 205→214★8 30→42★7                          │
│                                                                          │
│ (Reading·14) (Plan to read·9) (Completed·31) (On hold·2) (Dropped·4)     │
│ ▢ Omniscient Reader's Viewpoint   212/248 ▓▓▓▓░ +36 ★10 MU 8.9 3 mo ago  │
│   Jeonjijeok Dokja Sijeom · 전지적 독자 시점                               │
│ ▢ Return of the Mount Hua Sect     98/131 ▓▓▓░░ +33 ★9  MU 8.7 2 mo ago  │
│ ▢ Eleceed                         330/330 ▓▓▓▓▓     ★8  MU 8.8 4 d ago   │
│ …                                                                        │
└──────────────────────────────────────────────────────────────────────────┘
```

The "N new" counts on the type switch are *[my call]* so new chapters in a non-selected type aren't missed.
