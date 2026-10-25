// What the home screen shows for a chosen type and tab: pure functions, unit-tested.

export type TypeKey = 'manhwa' | 'manga' | 'manhua'
export type StatusKey = 'reading' | 'plan' | 'completed' | 'on_hold' | 'dropped'

export const TYPES: TypeKey[] = ['manhwa', 'manga', 'manhua']
export const TYPE_LABELS: Record<TypeKey, string> = {manhwa: 'Manhwa', manga: 'Manga', manhua: 'Manhua'}

export const STATUSES: StatusKey[] = ['reading', 'plan', 'completed', 'on_hold', 'dropped']
export const STATUS_LABELS: Record<StatusKey, string> = {
    reading: 'Reading',
    plan: 'Plan to read',
    completed: 'Completed',
    on_hold: 'On hold',
    dropped: 'Dropped',
}

/** The fields of a library item that view selection depends on. */
export interface ItemLike {
    effectiveType: string
    newCount: number
    series?: {title: string; muRating: number}
    entry: {status: string; myRating: number; lastReadAt: any; addedAt?: any; tags?: string[] | null}
    /** Marked 18+ by any source; hidden unless the owner allows 18+ titles. */
    adult?: boolean
}

export type ListItem = ItemLike & {series: {title: string; muRating: number}}

export type SortMode = 'smart' | 'rating' | 'title' | 'lastRead' | 'newCount' | 'muRating' | 'added'

export const SORT_MODES: SortMode[] = ['smart', 'rating', 'title', 'lastRead', 'newCount', 'muRating', 'added']
export const SORT_LABELS: Record<SortMode, string> = {
    smart: 'New chapters first',
    rating: 'Your rating',
    title: 'Title A to Z',
    lastRead: 'Recently read',
    newCount: 'Most new chapters',
    muRating: 'MangaUpdates rating',
    added: 'Recently added',
}

const ZERO_TIME = '0001-01-01T00:00:00Z'
const SIX_MONTHS_MS = 183 * 24 * 60 * 60 * 1000

function timeOf(iso: any): string {
    return !iso || String(iso).startsWith('0001-01-01') ? ZERO_TIME : String(iso)
}

/** Not read (or, if never read, not added) in the last 6 months. */
export function isStale(it: ItemLike, now: Date): boolean {
    const last = timeOf(it.entry.lastReadAt) !== ZERO_TIME ? it.entry.lastReadAt : it.entry.addedAt
    if (timeOf(last) === ZERO_TIME) return false
    return now.getTime() - new Date(last).getTime() > SIX_MONTHS_MS
}

const byTitle = (a: ItemLike, b: ItemLike) =>
    (a.series?.title ?? '').localeCompare(b.series?.title ?? '', undefined, {sensitivity: 'base', numeric: true})

const comparators: Record<SortMode, (a: ItemLike, b: ItemLike) => number> = {
    smart: (a, b) => compareItems(a, b),
    rating: (a, b) => ratingKey(b.entry.myRating) - ratingKey(a.entry.myRating) || byTitle(a, b),
    title: byTitle,
    lastRead: (a, b) => timeOf(b.entry.lastReadAt).localeCompare(timeOf(a.entry.lastReadAt)) || byTitle(a, b),
    newCount: (a, b) => b.newCount - a.newCount || ratingKey(b.entry.myRating) - ratingKey(a.entry.myRating),
    muRating: (a, b) => (b.series?.muRating ?? 0) - (a.series?.muRating ?? 0) || byTitle(a, b),
    added: (a, b) => timeOf(b.entry.addedAt).localeCompare(timeOf(a.entry.addedAt)) || byTitle(a, b),
}

/** A sorted copy of items. */
export function sortItems<T extends ItemLike>(items: T[], mode: SortMode): T[] {
    return [...items].sort(comparators[mode] ?? comparators.smart)
}

/** A series has new chapters worth surfacing only while the owner is reading it. */
export function hasNew(it: ItemLike): boolean {
    return (it.entry.status === 'reading' || it.entry.status === 'on_hold') && it.newCount > 0
}

function ratingKey(r: number): number {
    return r > 0 ? r : -1 // unrated sorts after every rating
}

/** New chapters first, then the owner's rating (unrated last), then most recently read. */
export function compareItems(a: ItemLike, b: ItemLike): number {
    const n = Number(hasNew(b)) - Number(hasNew(a))
    if (n !== 0) return n
    const r = ratingKey(b.entry.myRating) - ratingKey(a.entry.myRating)
    if (r !== 0) return r
    return String(b.entry.lastReadAt ?? '').localeCompare(String(a.entry.lastReadAt ?? ''))
}

const nativeScript = /[\p{Script=Hangul}\p{Script=Hiragana}\p{Script=Katakana}\p{Script=Han}]/u
const nativeFor: Record<string, RegExp> = {
    manhwa: /\p{Script=Hangul}/u,
    manga: /[\p{Script=Hiragana}\p{Script=Katakana}]/u,
    manhua: /^[^\p{Script=Hangul}\p{Script=Hiragana}\p{Script=Katakana}]*\p{Script=Han}/u,
}
const latinOnly = /^[\x20-\x7E’]+$/
const foreignArticle = /^(der|die|das|ein|eine|el|la|las|los|le|les|lo|il|gli|un|una|une|o|os|as)\s/i

/**
 * The alternate names worth showing under a title: one romanized name and the
 * native-script name (Korean for manhwa, Japanese for manga, Chinese for
 * manhua), rather than e.g. the German or Spanish translation.
 */
export function pickAltTitles(
    title: string,
    alts: string[] | undefined,
    type?: string,
    known: {romanized?: string; native?: string} = {},
    max = 2,
): string[] {
    const others = (alts ?? []).filter(a => a !== title)
    const usable = (t?: string) => (t && t !== title ? t : undefined)
    // MangaDex's language-tagged titles are exact. When they're known, show only
    // those; guessing the rest from the unlabeled list tends to pick translations.
    const knownTitles = [usable(known.romanized), usable(known.native)].filter((t): t is string => !!t)
    if (known.native || known.romanized) return knownTitles.slice(0, max)
    const latin = usable(known.romanized) ?? others.find(a => latinOnly.test(a) && !foreignArticle.test(a))
    const scriptRe = type ? nativeFor[type] : undefined
    const native = usable(known.native)
        ?? (scriptRe && others.find(a => scriptRe.test(a))) ?? others.find(a => nativeScript.test(a))
    const picked = [latin, native].filter((a): a is string => !!a)
    for (const a of others) {
        if (picked.length >= max) break
        if (!picked.includes(a)) picked.push(a)
    }
    return picked.slice(0, max)
}

export interface View<T> {
    shelf: T[]
    list: T[]
    newCounts: Record<TypeKey, number>
    tabCounts: Record<StatusKey, number>
    /** How many series in the selected type and tab carry each tag. */
    tagCounts: Record<string, number>
}

export interface ViewOptions {
    sort?: SortMode
    staleOnly?: boolean
    /** Only series with this tag (matched ignoring case); empty = all. */
    tag?: string
    /** Include 18+ series (off by default). */
    showAdult?: boolean
    now?: Date
}

/** The items the owner may see: 18+ series only when allowed. */
export function visibleItems<T extends ItemLike>(items: T[], showAdult: boolean): T[] {
    return showAdult ? items : items.filter(it => !it.adult)
}

export function selectView<T extends ItemLike>(items: T[], type: TypeKey, tab: StatusKey, opts: ViewOptions = {}): View<T> {
    const newCounts: Record<TypeKey, number> = {manhwa: 0, manga: 0, manhua: 0}
    const tabCounts: Record<StatusKey, number> = {reading: 0, plan: 0, completed: 0, on_hold: 0, dropped: 0}
    const tagCounts: Record<string, number> = {}
    const ofType: T[] = []
    const inTab: T[] = []
    for (const it of visibleItems(items, opts.showAdult ?? false)) {
        if (hasNew(it) && it.effectiveType in newCounts) newCounts[it.effectiveType as TypeKey]++
        if (it.effectiveType !== type) continue
        ofType.push(it)
        if (it.entry.status in tabCounts) tabCounts[it.entry.status as StatusKey]++
        if (it.entry.status !== tab) continue
        inTab.push(it)
        for (const t of it.entry.tags ?? []) tagCounts[t] = (tagCounts[t] ?? 0) + 1
    }
    const now = opts.now ?? new Date()
    const tag = (opts.tag ?? '').toLowerCase()
    const list = inTab.filter(it =>
        (!opts.staleOnly || isStale(it, now)) &&
        (!tag || (it.entry.tags ?? []).some(t => t.toLowerCase() === tag)))
    return {
        shelf: ofType.filter(hasNew).sort(compareItems),
        list: sortItems(list, opts.sort ?? 'smart'),
        newCounts,
        tabCounts,
        tagCounts,
    }
}
