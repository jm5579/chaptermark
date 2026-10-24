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
    entry: {status: string; myRating: number; lastReadAt: any}
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
    // MangaDex's language-tagged titles are exact; the rest is a best guess.
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
}

export function selectView<T extends ItemLike>(items: T[], type: TypeKey, tab: StatusKey): View<T> {
    const newCounts: Record<TypeKey, number> = {manhwa: 0, manga: 0, manhua: 0}
    const tabCounts: Record<StatusKey, number> = {reading: 0, plan: 0, completed: 0, on_hold: 0, dropped: 0}
    const ofType: T[] = []
    for (const it of items) {
        if (hasNew(it) && it.effectiveType in newCounts) newCounts[it.effectiveType as TypeKey]++
        if (it.effectiveType !== type) continue
        ofType.push(it)
        if (it.entry.status in tabCounts) tabCounts[it.entry.status as StatusKey]++
    }
    return {
        shelf: ofType.filter(hasNew).sort(compareItems),
        list: ofType.filter(it => it.entry.status === tab).sort(compareItems),
        newCounts,
        tabCounts,
    }
}
