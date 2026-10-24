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
