# ChapterMark v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build ChapterMark — a portable Windows exe that tracks the owner's last-read chapter per manga/manhwa/manhua, finds series by any alternate name via MangaUpdates, and shows newly translated chapters sorted by the owner's rating.

**Architecture:** Wails v2 app. A Go backend (`internal/...` packages: SQLite store, API clients, library service, background checker) is bound to a React + TypeScript frontend through Wails method bindings and events. All owner data lives in `data/` next to the exe.

**Tech Stack:** Go 1.27 (module min `go 1.23`), Wails v2.17.0, `modernc.org/sqlite` (pure Go), `golang.org/x/text`, React 18 + TypeScript 5 + Vite 5, Vitest.

**Spec:** `docs/design.md` (approved 2026-10-23). Executors read both.

## Global Constraints

- Module path `github.com/jm5579/chaptermark`; Wails CLI and library pinned to `v2.17.0`.
- Windows 10/11 only; UI rendered by WebView2. App/window title `ChapterMark`; exe `ChapterMark.exe`.
- Data dir = `<directory of the running exe>/data` — never `%APPDATA%`, never the working directory. Subdirs: `covers/`, `backups/`; DB file `library.db`.
- No API keys. Every outbound request sends `User-Agent: ChapterMark/1.0 (+https://github.com/jm5579/chaptermark)`.
- Politeness: MangaUpdates ≥ 1 s between requests; MangaDex ≥ 250 ms; AniList ≥ 2 s. Per-request timeout 15 s. Retry at most 3 times on HTTP 429/5xx, honouring `Retry-After`, else exponential back-off starting at 2 s.
- Owner rating: integer 1–10, `0` = unrated. MU rating shown /10 as-is; AniList `averageScore` /100 → /10, one decimal.
- Types: `manhwa` | `manga` | `manhua` (`other` only as a raw MU mapping; every tracked series must end up with one of the three via `type_override`). Statuses: `reading` | `plan` | `completed` | `on_hold` | `dropped`, labelled "Reading", "Plan to read", "Completed", "On hold", "Dropped".
- Defaults: auto-check interval 6 h; per-series details refresh every 7 days; keep 10 daily backups.
- Timestamps stored and sent as UTC RFC 3339 (`2026-10-23T14:05:00Z`). Zero time = never.
- Theme tokens: bg `#141218`, surface `#211e27`, surface-2 `#2a2632`, border `#33303b`, text `#ece8f1`, muted `#9b95a6`, accent `#ff8a5c`, new-badge `#ff6b35`, star `#ffcf5c`.
- Never commit `data/`, `*.db`, exports, `build/bin/` (already in `.gitignore`).
- Go is at `C:\Program Files\Go\bin` and Wails at `%USERPROFILE%\go\bin`; in Git Bash prefix commands with `export PATH="/c/Program Files/Go/bin:$HOME/go/bin:$PATH";`.

## Review Focus

1. **Folder not writable** (run from inside a zip or Program Files) → app shows a clear "move the ChapterMark folder somewhere you can write to" message instead of crashing. *Test in Task 2.*
2. **Offline at startup / API down** → library loads from SQLite instantly; check reports `offline`, and stored latest-chapter values are untouched. *Test in Task 12.*
3. **Unparseable or missing releases** (oneshots, "Prologue", series with zero releases) → latest stays unknown, no `+N` badge, never a negative count. *Tests in Tasks 3 and 8.*
4. **Search text with apostrophes, accents, native scripts, or `%`/`_`** → matches the right series, no SQL error, wildcards treated literally. *Test in Task 5.*
5. **MU type outside the three tabs** (OEL, Doujinshi…) added anyway → series must land in a visible tab (owner picks the type; defaults to `manga`). Also: adding a series twice must not duplicate it, and MangaDex/AniList failures must not fail the add. *Tests in Task 11.*

---

## File map

```
main.go                         Wails bootstrap, window options, asset + cover handler
app.go                          App struct: bound methods (thin), startup/shutdown, events
internal/paths/                 DataDir resolution + writability check
internal/chapters/              chapter-string parsing, new-count, effective latest
internal/textnorm/              title normalisation for local search
internal/store/                 SQLite open/migrate, series/entries/alt titles/meta CRUD
internal/backup/                daily VACUUM INTO backups + pruning
internal/httpx/                 shared HTTP client: UA, throttle, retry, JSON helpers
internal/sources/mangaupdates/  MU search, series detail, latest release (+ testdata/)
internal/sources/mangadex/      link lookup by MU id (base-36) (+ testdata/)
internal/sources/anilist/       score lookup (+ testdata/)
internal/covers/                cover download + /covers/ HTTP handler
internal/library/               service: add/update/remove/list/search/refresh
internal/checker/               background new-chapter check with progress
internal/exportimport/          JSON export/import (merge | replace)
frontend/src/lib/               view.ts (filter/sort/shelf/counts), time.ts — Vitest
frontend/src/components/        TopBar, TypeSwitch, Shelf, StatusTabs, SeriesRow, Cover,
                                SearchPanel, AddDialog, DetailPanel, SettingsDialog, RatingInput
frontend/src/styles.css         theme tokens + layout
.github/workflows/              ci.yml, release.yml
```

---

### Task 1: Scaffold the Wails app and CI

**Files:**
- Create (via `wails init -n chaptermark -t react-ts` in a temp dir, then move in): `main.go`, `app.go`, `wails.json`, `go.mod`, `go.sum`, `frontend/**`, `build/**`
- Create: `.gitattributes`, `.github/workflows/ci.yml`
- Modify: `.gitignore` (merge template entries; keep ours)

**Interfaces:** Produces `main.App` with `startup(ctx)`; module `github.com/jm5579/chaptermark`.

- [ ] **Step 1:** Init template in the scratchpad, copy everything except its `README.md`/`.gitignore` into the repo. Set `go.mod` module to `github.com/jm5579/chaptermark`, `go 1.23`, wails `v2.17.0`. In `wails.json`: `"name": "ChapterMark"`, `"outputfilename": "ChapterMark"`.
- [ ] **Step 2:** `main.go` window options: `Title: "ChapterMark"`, `Width: 1280, Height: 820, MinWidth: 1000, MinHeight: 680`, `BackgroundColour: &options.RGBA{R: 20, G: 18, B: 24, A: 1}`, `Windows: &windows.Options{Theme: windows.Dark}`.
- [ ] **Step 3:** Upgrade frontend deps: React 18.3, TypeScript 5, Vite 5, `@vitejs/plugin-react` 4; add `vitest` and script `"test": "vitest run"`. Replace template UI with an empty dark `<div id="app">ChapterMark</div>`.
- [ ] **Step 4:** `.gitattributes`: `* text=auto eol=lf`, `*.png binary`, `*.ico binary`.
- [ ] **Step 5:** `ci.yml` on push/PR, `windows-latest`: setup-go (`go-version-file: go.mod`), setup-node 24, `go test ./...`, `npm ci && npm test` in `frontend`, `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`, `wails build -clean`.
- [ ] **Step 6: Verify** `wails build -clean` → `build/bin/ChapterMark.exe` exists; launching it shows a dark window titled ChapterMark.
- [ ] **Step 7: Commit** `chore: scaffold Wails app, CI`.

### Task 2: Portable data directory

**Files:** Create `internal/paths/paths.go`, `internal/paths/paths_test.go`

**Interfaces:** Produces
- `func DataDirFor(exePath string) string` → `filepath.Join(filepath.Dir(exePath), "data")`
- `func Ensure(dataDir string) error` — creates `dataDir`, `covers/`, `backups/`; then writes and deletes a probe file. On failure returns `ErrNotWritable` wrapping the cause.
- `var ErrNotWritable = errors.New("ChapterMark can't save here. Move the ChapterMark folder somewhere you can write to (e.g. Desktop or Documents) and run it again.")`
- `func ExeDataDir() (string, error)` — `os.Executable()` + `filepath.EvalSymlinks` + `DataDirFor`.

- [ ] **Step 1: Failing tests:** `TestDataDirFor` (`C:\x\ChapterMark.exe` → `C:\x\data`); `TestEnsureCreatesSubdirs` (temp dir → `covers`, `backups` exist); `TestEnsureNotWritable` (dataDir path whose parent is a regular *file* → `errors.Is(err, ErrNotWritable)`).
- [ ] **Step 2:** `go test ./internal/paths/` → FAIL (undefined).
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `go test ./internal/paths/` → PASS.
- [ ] **Step 5: Commit** `feat(paths): resolve portable data dir next to exe`.

### Task 3: Chapter parsing and counts

**Files:** Create `internal/chapters/chapters.go`, `internal/chapters/chapters_test.go`

**Interfaces:** Produces
- `func Parse(raw string) (float64, bool)`
- `func NewCount(latest, mine float64) int`
- `func Effective(fetched float64, manual *float64) float64` — `max(fetched, *manual)`; `fetched` if manual nil.

Parse algorithm (not determined by the signature):
```
s := lowercase(raw)
remove volume/season tokens: regexp `\b(?:v|vol|volume|s|season)\.?\s*\d+(?:\.\d+)?`
nums := all matches of `\d+(?:\.\d+)?` in what remains
if none → (0, false); else → (max(nums), true)
```

- [ ] **Step 1: Failing table test `TestParse`:** `"311"→311`, `"10.5"→10.5`, `"1-5"→5`, `"c.45"→45`, `"v.3 c.12"→12`, `"v.15 c.3"→3`, `"Season 2 Ep. 4"→4`, `"S2 - 15"→15`, `"Vol.2 Ch.10-12"→12`, `"c.100 (end)"→100`; not ok: `""`, `"Oneshot"`, `"Prologue"`, `"v.3"`.
  `TestNewCount`: `(248,212)→36`, `(212,212)→0`, `(0,50)→0`, `(10.5,10)→0`, `(11,10.5)→1`, `(100,120)→0`.
  `TestEffective`: `(311,nil)→311`, `(311,&320)→320`, `(330,&320)→330`.
- [ ] **Step 2:** `go test ./internal/chapters/` → FAIL.
- [ ] **Step 3:** Implement; `NewCount` = `max(0, floor(latest) − floor(mine))`, and 0 when `latest <= 0`.
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(chapters): parse release chapter strings`.

### Task 4: Title normalisation

**Files:** Create `internal/textnorm/textnorm.go`, `internal/textnorm/textnorm_test.go`

**Interfaces:** Produces `func Normalize(s string) string` — NFKD → drop `unicode.Mn` → NFC → lowercase → every rune that is not a letter/number becomes a space → collapse spaces → trim; apostrophes (`'`, `’`) are *removed* (not spaced) so "Reader's" → "readers".

- [ ] **Step 1: Failing table test `TestNormalize`:** `"Omniscient Reader's Viewpoint"→"omniscient readers viewpoint"`, `"Pokémon"→"pokemon"`, `"  Solo   Leveling: Ragnarok! "→"solo leveling ragnarok"`, `"전지적 독자 시점"→"전지적 독자 시점"`, `"ＯＲＶ"→"orv"`, `"100%_done"→"100 done"`.
- [ ] **Step 2–4:** fail → implement with `golang.org/x/text/unicode/norm` → pass (`go test ./internal/textnorm/`).
- [ ] **Step 5: Commit** `feat(textnorm): normalise titles for search`.

### Task 5: SQLite store

**Files:** Create `internal/store/store.go`, `internal/store/migrations.go`, `internal/store/series.go`, `internal/store/meta.go`, `internal/store/store_test.go`

**Interfaces:** Consumes `textnorm.Normalize`. Produces:
```go
type SeriesType string // "manhwa","manga","manhua","other"
type Status string     // "reading","plan","completed","on_hold","dropped"

type Series struct {
    ID, MUID int64; MDID string; ALID, MALID int
    Title, MUURL, Year, PubStatus string; Type, TypeOverride SeriesType; Completed bool
    CoverFile, CoverURL string
    MURating float64; MUVotes int; ALRating float64
    LatestChapter float64; LatestRaw, LatestGroup, LatestDate string; LatestManual *float64
    DetailsRefreshedAt, ChaptersCheckedAt time.Time
    AltTitles []string
}
type Entry struct {
    SeriesID int64; Status Status; MyChapter float64; MyRating int
    Notes, ReadURL string; LastReadAt, AddedAt, UpdatedAt time.Time
}
type Item struct { Series Series `json:"series"`; Entry Entry `json:"entry"` }
type Latest struct { Chapter float64; Raw, Group, Date string }

func Open(path string) (*Store, error)          // WAL, foreign_keys, busy_timeout=5000; does NOT migrate
func (s *Store) Migrate(ctx context.Context) error
func (s *Store) DB() *sql.DB
func (s *Store) Close() error
func (s *Store) UpsertSeries(ctx context.Context, sr *Series) error   // by MUID; sets sr.ID; replaces alt titles (title always included)
func (s *Store) SeriesByMUID(ctx context.Context, muID int64) (Series, bool, error)
func (s *Store) GetItem(ctx context.Context, seriesID int64) (Item, error)
func (s *Store) ListItems(ctx context.Context) ([]Item, error)
func (s *Store) SaveEntry(ctx context.Context, e Entry) error          // upsert by SeriesID
func (s *Store) DeleteSeries(ctx context.Context, id int64) error      // cascades entry + alt titles
func (s *Store) SearchLocal(ctx context.Context, query string, limit int) ([]Item, error)
func (s *Store) SetLatest(ctx context.Context, id int64, l Latest, checkedAt time.Time) error // clears latest_manual when l.Chapter > it
func (s *Store) SetLatestManual(ctx context.Context, id int64, v *float64) error
func (s *Store) GetMeta(ctx context.Context, key string) (string, bool, error)
func (s *Store) SetMeta(ctx context.Context, key, value string) error
```
Schema v1 (tables `series`, `alt_titles(series_id, title, normalized)` with index on `normalized`, `entry` PK `series_id`, `meta(key PK, value)`), versioned with `PRAGMA user_version`. `SearchLocal` normalises the query and matches `normalized LIKE '%' || ? || '%' ESCAPE '\'` after escaping `\`, `%`, `_`; returns distinct items ordered by title.

- [ ] **Step 1: Failing tests** (temp-dir DB): `TestMigrateIdempotent` (twice → `user_version` 1); `TestUpsertSeriesByMUID` (second upsert same MUID keeps ID, replaces alt titles); `TestSaveAndListItems` (round-trip all fields incl. `LatestManual` nil/non-nil and times); `TestDeleteCascades`; `TestSearchLocalAltNames` — series "Omniscient Reader" with alts `["Jeonjijeok Dokja Sijeom","ORV","전지적 독자 시점"]`: queries `"jeonjijeok"`, `"orv"`, `"전지적"`, `"omniscient reader"` all hit; `TestSearchLocalLiteralWildcards` — `"%"` and `"_"` return nothing (no error) when no title contains them, and `"100%"` matches only a title containing "100%"; `TestSetLatestClearsManual`.
- [ ] **Step 2:** `go test ./internal/store/` → FAIL.
- [ ] **Step 3:** Implement (driver name `"sqlite"`, DSN `file:<path>?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)`).
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(store): SQLite schema, migrations, CRUD, local search`.

### Task 6: Daily backups

**Files:** Create `internal/backup/backup.go`, `internal/backup/backup_test.go`

**Interfaces:** Produces `func Daily(db *sql.DB, dir string, keep int, now time.Time) (created bool, err error)` — writes `library-YYYY-MM-DD.db` via `VACUUM INTO` if today's file doesn't exist; then deletes the oldest `library-*.db` beyond `keep`.

- [ ] **Step 1: Failing tests:** `TestDailyCreatesOncePerDay` (second call same day → `created=false`); `TestDailyPrunes` (12 pre-existing dated files, keep 10 → 10 remain, newest kept); backup opens as valid SQLite containing the source rows.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** `feat(backup): daily SQLite backups with pruning`.

### Task 7: Shared HTTP client

**Files:** Create `internal/httpx/httpx.go`, `internal/httpx/httpx_test.go`

**Interfaces:** Produces
```go
const UserAgent = "ChapterMark/1.0 (+https://github.com/jm5579/chaptermark)"
func New(minInterval time.Duration) *Client
func (c *Client) GetJSON(ctx context.Context, url string, out any) error
func (c *Client) PostJSON(ctx context.Context, url string, body, out any) error
func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, string /*content-type*/, error)
var ErrOffline = errors.New("offline")   // returned (wrapped) for DNS/connection errors
type StatusError struct{ Code int; URL string }
```
Field `c.Sleep func(time.Duration)` (defaults to `time.Sleep`) and `c.BaseBackoff` (default 2 s) so tests run fast.

- [ ] **Step 1: Failing tests** with `httptest.Server`: `TestSendsUserAgent`; `TestThrottleSpacing` (3 calls with minInterval 50 ms take ≥ 100 ms); `TestRetriesOn429ThenSucceeds` (`Retry-After: 0`); `TestGivesUpAfter3Retries` (always 503 → `*StatusError` 503, 4 attempts); `TestOfflineError` (closed server → `errors.Is(err, ErrOffline)`).
- [ ] **Step 2–4:** fail → implement (one mutex-guarded "next allowed time" per client; 15 s `http.Client.Timeout`) → pass.
- [ ] **Step 5: Commit** `feat(httpx): polite HTTP client with retries`.

### Task 8: MangaUpdates client

**Files:** Create `internal/sources/mangaupdates/client.go`, `.../client_test.go`, `.../testdata/{search_orv.json, series_orv.json, releases_orv.json, releases_empty.json}` (record live responses for series `50369844984`; `releases_empty.json` = `{"total_hits":0,"results":[]}`)

**Interfaces:** Consumes `httpx.Client`, `chapters.Parse`. Produces
```go
type Client struct{ HTTP *httpx.Client; BaseURL string } // default "https://api.mangaupdates.com/v1"
type SearchResult struct{ ID int64; Title, HitTitle, Type, Year, URL, CoverURL string; Rating float64 }
type SeriesDetail struct{ ID int64; Title, URL, Type, Year, Status string; Completed bool; Rating float64; Votes int; CoverURL string; AltTitles []string }
type Release struct{ Chapter float64; Raw, Group, Date string }
func (c *Client) Search(ctx context.Context, q string, page int) ([]SearchResult, int, error)   // POST /series/search {"search","page","perpage":25}
func (c *Client) Series(ctx context.Context, id int64) (SeriesDetail, error)                    // GET /series/{id}
func (c *Client) LatestRelease(ctx context.Context, id int64) (Release, bool, error)
func MapType(muType string) store.SeriesType  // "Manhwa"→manhwa, "Manga"→manga, "Manhua"→manhua, else other
```
`LatestRelease`: POST `/releases/search` `{"search":"<id>","search_type":"series","orderby":"date","asc":"desc","page":1}`; **do not use `orderby: chap`** (MU sorts it as text: "99" > "311"). Parse every result's `chapter`, return the highest; ties keep the first (most recent). `Group` = group names joined with `" & "`. `CoverURL` = `image.url.original`.

- [ ] **Step 1: Failing tests** against an httptest server serving the fixtures: `TestSearchFindsORVByKoreanName` (first result ID `50369844984`, Type `"Manhwa"`, Rating `8.82`, HitTitle `"Jeonjijeok Dokja Sijeom"`); `TestSeriesDetail` (≥ 8 alt titles incl. `"ORV"`, Completed false); `TestLatestReleasePicksHighest` (Chapter `311`, Group contains `"Flame Comics"`, Date `"2026-05-19"`); `TestLatestReleaseNone` (empty → ok=false, no error); `TestMapType`; request-body assertion that `orderby` = `"date"`.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** `feat(mangaupdates): search, details, latest release`.

### Task 9: MangaDex and AniList clients

**Files:** Create `internal/sources/mangadex/client.go`, `.../client_test.go`, `.../testdata/search_orv.json`; `internal/sources/anilist/client.go`, `.../client_test.go`, `.../testdata/media_119257.json`

**Interfaces:** Produces
```go
// mangadex
type Links struct{ MDID string; ALID, MALID int; CoverURL string }
func MUIDToBase36(id int64) string                    // 50369844984 → "n50wl4o"
func (c *Client) FindByMU(ctx context.Context, title string, muID int64) (Links, bool, error)
// GET /manga?title=<q>&limit=10&includes[]=cover_art ; match attributes.links.mu == MUIDToBase36(muID)
// CoverURL = https://uploads.mangadex.org/covers/<mangaID>/<fileName>.512.jpg
// anilist
func (c *Client) Score(ctx context.Context, id int) (float64, bool, error) // averageScore/10 rounded to 1 dp; ok=false when null
```
Both clients have `HTTP *httpx.Client; BaseURL string` (defaults `https://api.mangadex.org`, `https://graphql.anilist.co`).

- [ ] **Step 1: Failing tests:** `TestMUIDToBase36`; `TestFindByMUMatchesLink` (Links `{MDID:"9a414441-bbad-43f1-a3a7-dc262ca790a3", ALID:119257, MALID:132214}`); `TestFindByMUNoMatch` (muID 1 → ok=false); `TestScore` (`8.6`); `TestScoreNull` (`averageScore: null` → ok=false).
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** `feat(sources): MangaDex linking and AniList score`.

### Task 10: Covers

**Files:** Create `internal/covers/covers.go`, `internal/covers/covers_test.go`

**Interfaces:** Produces
- `func Download(ctx context.Context, c *httpx.Client, url, dir string, seriesID int64) (string, error)` — saves `<seriesID>.<jpg|png|webp>` (from content-type; default jpg), removing any older file for that id; returns the file name.
- `func Handler(dir string) http.Handler` — serves `GET /covers/<name>`; rejects names that aren't `^\d+\.(jpg|png|webp)$` with 404; sets `Cache-Control: max-age=86400`.

- [ ] **Step 1: Failing tests:** `TestDownloadSavesByContentType` (png server → `"7.png"`); `TestDownloadReplacesOldExtension` (existing `7.jpg` removed); `TestHandlerServesFile`; `TestHandlerRejectsTraversal` (`/covers/..%2Flibrary.db` → 404).
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** `feat(covers): download and serve cached covers`.

### Task 11: Library service

**Files:** Create `internal/library/library.go`, `internal/library/view.go`, `internal/library/library_test.go`

**Interfaces:** Consumes Tasks 3–10. Produces
```go
type MU interface {
    Search(ctx context.Context, q string, page int) ([]mangaupdates.SearchResult, int, error)
    Series(ctx context.Context, id int64) (mangaupdates.SeriesDetail, error)
    LatestRelease(ctx context.Context, id int64) (mangaupdates.Release, bool, error)
}
type MD interface{ FindByMU(ctx context.Context, title string, muID int64) (mangadex.Links, bool, error) }
type AL interface{ Score(ctx context.Context, id int) (float64, bool, error) }
type CoverFetcher func(ctx context.Context, url string, seriesID int64) (string, error)

type Service struct{ Store *store.Store; MU MU; MD MD; AL AL; FetchCover CoverFetcher; Now func() time.Time }

type ItemView struct {
    store.Item
    EffectiveType store.SeriesType `json:"effectiveType"`
    EffectiveLatest float64        `json:"effectiveLatest"`
    NewCount int                   `json:"newCount"`
}
type OnlineResult struct{ mangaupdates.SearchResult; MappedType store.SeriesType `json:"mappedType"`; InLibrary bool `json:"inLibrary"`; SeriesID int64 `json:"seriesId"` }
type AddInput struct{ Status store.Status; MyChapter float64; MyRating int; TypeOverride store.SeriesType }
type EntryUpdate struct{ Status store.Status; MyChapter float64; MyRating int; Notes, ReadURL string; TypeOverride store.SeriesType; LatestManual *float64 }

func (s *Service) List(ctx context.Context) ([]ItemView, error)
func (s *Service) Get(ctx context.Context, seriesID int64) (ItemView, error)
func (s *Service) SearchLocal(ctx context.Context, q string) ([]ItemView, error)          // limit 20
func (s *Service) SearchOnline(ctx context.Context, q string, page int, includeOther bool) ([]OnlineResult, error)
func (s *Service) Add(ctx context.Context, muID int64, in AddInput) (ItemView, error)
func (s *Service) UpdateEntry(ctx context.Context, seriesID int64, in EntryUpdate) (ItemView, error)
func (s *Service) Remove(ctx context.Context, seriesID int64) error
func (s *Service) RefreshDetails(ctx context.Context, seriesID int64) error   // MU details + alt titles + MD links + AL score + cover; sets DetailsRefreshedAt
func (s *Service) RefreshLatest(ctx context.Context, seriesID int64) error    // MU LatestRelease → store.SetLatest
func (s *Service) DetailsStale(it store.Item) bool                            // DetailsRefreshedAt older than 7 days
```
`EffectiveType` = `TypeOverride` if set, else `Type`. `Add` on an existing MUID returns the existing item unchanged. `Add` when MU type maps to `other` and `in.TypeOverride` is empty stores `TypeOverride = manga`. `Add` sets `AddedAt`/`UpdatedAt`, and `LastReadAt` only when `MyChapter > 0`. `UpdateEntry` sets `LastReadAt = Now()` only when `MyChapter` changed. MD/AL/cover errors are swallowed (logged); MU errors fail the call.

- [ ] **Step 1: Failing tests** with in-package fakes for MU/MD/AL and a temp store: `TestAddFetchesEverything` (alt titles stored, ALRating 8.6, cover file set, latest 311); `TestAddTwiceNoDuplicate`; `TestAddOtherTypeLandsInVisibleTab` (MU type `"OEL"` → EffectiveType `manga`; with override `manhwa` → `manhwa`); `TestAddSurvivesMDandALFailure` (MD/AL return errors → add succeeds, ALRating 0); `TestUpdateEntryTouchesLastReadOnlyOnChapterChange`; `TestNewCountAndEffectiveLatest` (manual 320 over fetched 311 → EffectiveLatest 320, my 300 → NewCount 20); `TestSearchOnlineFlagsInLibraryAndHidesOther` (`includeOther=false` drops Novel/Doujinshi results).
- [ ] **Step 2–4:** fail → implement → pass (`go test ./internal/library/`).
- [ ] **Step 5: Commit** `feat(library): add/update/search service`.

### Task 12: Background checker

**Files:** Create `internal/checker/checker.go`, `internal/checker/checker_test.go`

**Interfaces:** Consumes `library.Service`, `store.Store`, `httpx.ErrOffline`. Produces
```go
type Progress struct {
    Running bool `json:"running"`; Done, Total, Failed int `json:"..."`
    LastCheckedAt time.Time `json:"lastCheckedAt"`; Error string `json:"error"` // "" | "offline" | "partial"
}
type Checker struct{ Lib *library.Service; Store *store.Store; Interval func() time.Duration; Now func() time.Time }
func (c *Checker) Due(ctx context.Context) (bool, error)        // meta "last_full_check_at" older than Interval() (or missing)
func (c *Checker) Run(ctx context.Context, emit func(Progress), onItem func(library.ItemView)) error
func (c *Checker) Status() Progress
```
`Run`: no-op if already running (atomic flag). Checks entries with status `reading`, `on_hold`, `plan`, ordered by `MyRating` desc, then `LastReadAt` desc. Per series: `RefreshLatest`; also `RefreshDetails` when `DetailsStale`. Emits progress after each series and calls `onItem` with the updated view. Stops early if the first 3 consecutive failures are all `ErrOffline` → `Error = "offline"`. Writes `last_full_check_at` only if ≥ 1 series succeeded. `Error = "partial"` when some failed.

- [ ] **Step 1: Failing tests** (fake MU via `library.Service` with fakes from Task 11 — export them as `library/libtest` package if needed): `TestRunOrdersByRating`; `TestRunSkipsCompletedAndDropped`; `TestOfflineKeepsStoredLatest` (fake MU returns wrapped `httpx.ErrOffline` → `Error=="offline"`, stored latest unchanged, `last_full_check_at` unchanged); `TestDue` (missing → true; 5 h ago with 6 h interval → false; 7 h → true); `TestRunNotReentrant`.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** `feat(checker): background new-chapter check`.

### Task 13: Export / import

**Files:** Create `internal/exportimport/exportimport.go`, `internal/exportimport/exportimport_test.go`

**Interfaces:** Produces
```go
type Mode string // "merge" | "replace"
type File struct{ App string `json:"app"`; Version int `json:"version"`; ExportedAt time.Time `json:"exportedAt"`; Items []store.Item `json:"items"` }
type Result struct{ Added, Updated, Skipped int }
func Export(ctx context.Context, st *store.Store, w io.Writer, now time.Time) error // App "ChapterMark", Version 1, indented JSON
func Import(ctx context.Context, st *store.Store, r io.Reader, mode Mode) (Result, error)
```
Rejects files whose `App != "ChapterMark"` or `Version > 1`. Replace = delete all series then insert. Merge = insert unknown MUIDs; for known MUIDs keep the entry with the later `UpdatedAt`. Imported series get `DetailsRefreshedAt` zero so the checker refreshes covers/ratings.

- [ ] **Step 1: Failing tests:** `TestRoundTrip` (export → replace-import into fresh store → equal entries); `TestMergeKeepsNewer`; `TestRejectsForeignFile`.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** `feat(exportimport): JSON export and import`.

### Task 14: Wire the backend into Wails

**Files:** Modify `main.go`, `app.go`; Create `app_test.go`

**Interfaces:** Produces bound methods (generated into `frontend/wailsjs/go/main/App`):
```go
func (a *App) Init() InitResult   // InitResult{Ready bool; Error string; DataDir string; Version string}; main.go declares `var version = "dev"` (overridden by -ldflags)
func (a *App) ListLibrary() ([]library.ItemView, error)
func (a *App) SearchLocal(q string) ([]library.ItemView, error)
func (a *App) SearchOnline(q string, page int, includeOther bool) ([]library.OnlineResult, error)
func (a *App) AddSeries(muID int64, in library.AddInput) (library.ItemView, error)
func (a *App) UpdateEntry(seriesID int64, in library.EntryUpdate) (library.ItemView, error)
func (a *App) RemoveSeries(seriesID int64) error
func (a *App) CheckNow()
func (a *App) CheckStatus() checker.Progress
func (a *App) GetSettings() Settings          // Settings{CheckIntervalHours int; BackupsKeep int}
func (a *App) SaveSettings(s Settings) error   // validates 1–168 h, 1–50 backups
func (a *App) ExportLibrary() (string, error)  // SaveFileDialog, default "chaptermark-export-YYYY-MM-DD.json"; "" if cancelled
func (a *App) ImportLibrary(mode string) (exportimport.Result, error) // OpenFileDialog; cancelled → zero Result
func (a *App) OpenDataFolder() error
func (a *App) OpenURL(url string)              // http/https only
```
Events emitted: `"check:progress"` (Progress), `"library:item"` (ItemView). Startup order: `paths.ExeDataDir` → `paths.Ensure` → `store.Open` → `backup.Daily` (keep from settings) → `Migrate` → build clients/service/checker → if `Due`, `go CheckNow()`. Any failure → `Init()` returns `Ready:false` with the message (ErrNotWritable text verbatim). Window size/position saved to meta key `window` on shutdown, restored on startup. `main.go`: `AssetServer: &assetserver.Options{Assets: assets, Handler: coverHandler}` where `coverHandler` lazily serves `covers.Handler(dataDir/covers)`.

- [ ] **Step 1: Failing test** `TestSaveSettingsValidates` (0 h → error; 200 backups → error; 6/10 → ok and persisted).
- [ ] **Step 2–4:** fail → implement → pass; `wails build` succeeds and `frontend/wailsjs/go/models.ts` contains `ItemView`.
- [ ] **Step 5: Commit** `feat(app): bind backend to Wails`.

### Task 15: Frontend foundation — theme and view logic

**Files:** Create `frontend/src/styles.css`, `frontend/src/lib/view.ts`, `frontend/src/lib/view.test.ts`, `frontend/src/lib/time.ts`, `frontend/src/lib/time.test.ts`, `frontend/src/lib/api.ts`

**Interfaces:** Consumes generated `wailsjs` models. Produces
```ts
export type TypeKey = 'manhwa' | 'manga' | 'manhua'
export type StatusKey = 'reading' | 'plan' | 'completed' | 'on_hold' | 'dropped'
export const STATUS_LABELS: Record<StatusKey, string>   // spec labels
export function hasNew(it: ItemView): boolean            // status reading|on_hold && newCount > 0
export function compareItems(a: ItemView, b: ItemView): number // hasNew first, myRating desc (0 last), lastReadAt desc
export function selectView(items: ItemView[], type: TypeKey, tab: StatusKey):
  { shelf: ItemView[]; list: ItemView[]; newCounts: Record<TypeKey, number>; tabCounts: Record<StatusKey, number> }
export function relativeTime(iso: string, now: Date): string // "just now","5 min ago","3 h ago","4 days ago","2 weeks ago","3 months ago","1 year ago","never"(zero time)
export function formatChapter(n: number): string           // 311 → "311", 10.5 → "10.5", 0 → "–"
```
`api.ts` re-exports the bound functions plus `onEvent(name, cb)` wrappers. `styles.css` defines the Global Constraints tokens as CSS variables and base dark layout (system font stack `"Segoe UI Variable", "Segoe UI", system-ui`).

- [ ] **Step 1: Failing Vitest tests:** shelf contains only selected-type reading/on_hold items with newCount > 0 in rating order; plan items never on shelf; unrated sorts after rated; `newCounts` counts per type independent of the selected type; `tabCounts` per status within type; `relativeTime` table; `formatChapter` table.
- [ ] **Step 2:** `npm test` → FAIL. **Step 3:** implement. **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(ui): theme tokens and view selection logic`.

### Task 16: Home screen

**Files:** Create `frontend/src/components/{TopBar,TypeSwitch,Shelf,StatusTabs,SeriesRow,Cover,Toast}.tsx`; Modify `frontend/src/App.tsx`

**Interfaces:** Consumes Task 15 + bindings. `App` holds `items`, `type` (default `'manhwa'`), `tab` (default `'reading'`), `progress`; loads via `Init()` then `ListLibrary()`; subscribes to `library:item` (replace by `series.id`) and `check:progress`.

Behaviour (matches approved mockup `docs/mockups/home-layout-c.html`):
- TopBar: logo "CHAPTERMARK", search input (opens SearchPanel, Task 17), check button: `↻ Checked 2 min ago` / `Checking 34/120…` / `Couldn't check — offline · last 3 h ago`; settings gear.
- TypeSwitch: three segments with `N new` pill when > 0.
- Shelf (hidden when empty): cover cards with `+N` badge, `last → latest`, `★ rating`; card width adapts (auto-fill, min 150 px).
- StatusTabs with counts; rows: cover thumb, title, first two alt titles, progress bar (`my / latest`), `+N` badge only when `hasNew`, plan rows show `311 ch`, `★ n` or `–`, `MU 8.8`, `relativeTime(lastReadAt)`.
- Empty state for a type/tab: "Nothing here yet — search above to add a series."
- `Init().Ready === false` → full-screen message with the error text and the data folder path.
- Clicking a card/row opens DetailPanel (Task 18).

- [ ] **Step 1:** Implement components. **Step 2: Verify** `npm test` still passes; `wails dev` with a seeded DB shows layout C (type switch, shelf, tabs, rows) matching the mockup; empty DB shows empty state.
- [ ] **Step 3: Commit** `feat(ui): home screen (layout C)`.

### Task 17: Search and add

**Files:** Create `frontend/src/components/{SearchPanel,AddDialog,RatingInput}.tsx`; Modify `TopBar.tsx`, `App.tsx`

**Interfaces:** Consumes `SearchLocal`, `SearchOnline`, `AddSeries`. `RatingInput` props `{ value: number; onChange(v: number): void }` (1–10 + clear).

Behaviour: dropdown panel under the search box. "In your list" section updates on every keystroke (`SearchLocal`). "Add from MangaUpdates" section debounced 400 ms (`SearchOnline(q, 1, showOther)`), shows cover, title, matched alt name ("matched: Jeonjijeok Dokja Sijeom" when `hitTitle !== title`), type pill, year, MU rating; "In your list" chip instead of Add for `inLibrary`. "Show other types" checkbox. `Esc` closes. Add opens AddDialog: status (default Reading), current chapter (default 0), rating (optional), and — only when `mappedType === 'other'` — a required Manhwa/Manga/Manhua choice (default Manga). Confirm → `AddSeries` → item appears, switch jumps to its type, toast "Added <title>". Errors → inline message ("Couldn't reach MangaUpdates — check your internet").

- [ ] **Step 1:** Implement. **Step 2: Verify** in `wails dev`: typing "jeonjijeok dokja" finds Omniscient Reader; adding it shows cover, latest 311+, alt names; re-searching shows "In your list".
- [ ] **Step 3: Commit** `feat(ui): search and add series`.

### Task 18: Series detail panel

**Files:** Create `frontend/src/components/DetailPanel.tsx`; Modify `App.tsx`

**Interfaces:** Consumes `UpdateEntry`, `RemoveSeries`, `OpenURL`.

Behaviour: right-side slide-in panel (≈ 420 px, `Esc`/backdrop closes). Shows cover, title, all alt titles, type (editable select → `TypeOverride`), year, MU publication status text. **My chapter**: `−1` / number input (step 0.5 allowed) / `+1`. Latest: `311 · Flame Comics · 19 May 2026` + "set manually" input (→ `LatestManual`, clear button). Status select, RatingInput, notes textarea, "Where I read it" URL with an open button. Public ratings: `MangaUpdates 8.8 (949 votes)`, `AniList 8.6`. "Open on MangaUpdates". Remove (confirm dialog). Saves on change (chapter buttons immediately; text fields on blur), updating the list in place.

- [ ] **Step 1:** Implement. **Step 2: Verify:** `+1` updates progress bar, `+N` badge and "read just now"; moving to Completed removes it from the shelf; manual latest 400 shows `+N` accordingly; remove asks first.
- [ ] **Step 3: Commit** `feat(ui): series detail panel`.

### Task 19: Settings

**Files:** Create `frontend/src/components/SettingsDialog.tsx`; Modify `TopBar.tsx`, `App.tsx`

**Interfaces:** Consumes `GetSettings`, `SaveSettings`, `ExportLibrary`, `ImportLibrary`, `OpenDataFolder`.

Behaviour: modal with auto-check interval (hours), backups to keep, "Export list…", "Import list…" (asks Merge vs Replace; Replace needs a second "This replaces your whole list" confirm), "Open data folder", and the data folder path + app version.

- [ ] **Step 1:** Implement. **Step 2: Verify:** export then replace-import restores the same list; invalid interval shows the backend's validation message.
- [ ] **Step 3: Commit** `feat(ui): settings, export/import`.

### Task 20: Icon, release workflow, docs, final verification

**Files:** Create `build/appicon.png` (1024², generated from `build/appicon.svg` — dark rounded square, orange bookmark ribbon with a check), `.github/workflows/release.yml`; Modify `README.md` (remove "under construction", add screenshot), `docs/design.md` (nothing unless behaviour changed)

- [ ] **Step 1:** Generate the icon (one-off `npx sharp-cli` or similar; don't add runtime deps); delete `build/windows/icon.ico` so Wails regenerates it.
- [ ] **Step 2:** `release.yml` on tags `v*`: windows-latest, same setup as CI, `wails build -clean -ldflags "-X main.version=${{ github.ref_name }}"`, assemble `ChapterMark/ChapterMark.exe` → `ChapterMark-${{ github.ref_name }}-windows.zip`, `gh release create` with the zip.
- [ ] **Step 3: Full verification:** `go test ./...` PASS, `npm test` PASS, `wails build -clean` OK. Manual: copy `build/bin/ChapterMark.exe` into an empty folder → first run creates `data/`; add 3 series of different types; close; disconnect network (or block hosts) and reopen → list loads, check shows offline; copy the folder elsewhere → reopens with same data; check `data/backups/` has today's file.
- [ ] **Step 4: Commit** `chore: icon, release workflow, docs`; push branch; open PR to `main`.
