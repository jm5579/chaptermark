import {describe, expect, it} from 'vitest'
import {isStale, selectView, sortItems, type ListItem} from './view'

const ZERO = '0001-01-01T00:00:00Z'
let nextId = 1
function li(title: string, o: {
    rating?: number, newCount?: number, status?: string, lastRead?: string, added?: string,
    mu?: number, tags?: string[] | null, type?: string,
} = {}): ListItem & {id: number} {
    return {
        id: nextId++,
        effectiveType: o.type ?? 'manhwa',
        newCount: o.newCount ?? 0,
        series: {title, muRating: o.mu ?? 0},
        entry: {
            status: o.status ?? 'reading', myRating: o.rating ?? 0,
            lastReadAt: o.lastRead ?? ZERO, addedAt: o.added ?? ZERO, tags: o.tags ?? null,
        },
    }
}
const titles = (xs: ListItem[]) => xs.map(x => x.series.title)
const now = new Date('2026-10-24T12:00:00Z')

describe('sortItems', () => {
    const a = li('Eleceed', {rating: 8, lastRead: '2026-10-20T00:00:00Z', added: '2026-01-01T00:00:00Z', mu: 8.6})
    const b = li('Omniscient Reader', {rating: 10, newCount: 35, lastRead: '2026-07-01T00:00:00Z', added: '2026-03-01T00:00:00Z', mu: 8.8})
    const c = li('nano Machine', {rating: 0, newCount: 21, lastRead: ZERO, added: '2026-09-01T00:00:00Z', mu: 0})
    const all = [a, b, c]

    it.each([
        ['smart', ['Omniscient Reader', 'nano Machine', 'Eleceed']],
        ['rating', ['Omniscient Reader', 'Eleceed', 'nano Machine']],
        ['title', ['Eleceed', 'nano Machine', 'Omniscient Reader']],
        ['lastRead', ['Eleceed', 'Omniscient Reader', 'nano Machine']],
        ['newCount', ['Omniscient Reader', 'nano Machine', 'Eleceed']],
        ['muRating', ['Omniscient Reader', 'Eleceed', 'nano Machine']],
        ['added', ['nano Machine', 'Omniscient Reader', 'Eleceed']],
    ] as const)('%s', (mode, want) => {
        expect(titles(sortItems(all, mode))).toEqual(want)
    })

    it('does not change the input order', () => {
        const input = [a, b, c]
        sortItems(input, 'title')
        expect(input).toEqual([a, b, c])
    })
})

describe('isStale', () => {
    it('is true when last read more than 6 months ago', () => {
        expect(isStale(li('x', {lastRead: '2026-03-01T00:00:00Z'}), now)).toBe(true)
        expect(isStale(li('x', {lastRead: '2026-06-01T00:00:00Z'}), now)).toBe(false)
    })
    it('falls back to the date added for series never read', () => {
        expect(isStale(li('x', {added: '2025-12-01T00:00:00Z'}), now)).toBe(true)
        expect(isStale(li('x', {added: '2026-10-01T00:00:00Z'}), now)).toBe(false)
        expect(isStale(li('x'), now)).toBe(false)
    })
})

describe('selectView filters', () => {
    const fav = li('Fav', {rating: 9, tags: ['Favourites']})
    const both = li('Both', {rating: 8, tags: ['Favourites', 'Binge later'], lastRead: '2025-01-01T00:00:00Z'})
    const none = li('None', {rating: 7, lastRead: '2025-02-01T00:00:00Z'})
    const manga = li('Manga one', {type: 'manga', tags: ['Favourites']})
    const all = [fav, both, none, manga]

    it('filters by tag, ignoring case', () => {
        const v = selectView(all, 'manhwa', 'reading', {tag: 'favourites', now})
        expect(titles(v.list)).toEqual(['Fav', 'Both'])
    })
    it('counts tags within the selected type and tab', () => {
        const v = selectView(all, 'manhwa', 'reading', {now})
        expect(v.tagCounts).toEqual({'Favourites': 2, 'Binge later': 1})
    })
    it('shows only stale series when asked', () => {
        const v = selectView(all, 'manhwa', 'reading', {staleOnly: true, now})
        expect(titles(v.list)).toEqual(['Both', 'None'])
    })
    it('applies the chosen sort to the list but not to the shelf', () => {
        const v = selectView(all, 'manhwa', 'reading', {sort: 'title', now})
        expect(titles(v.list)).toEqual(['Both', 'Fav', 'None'])
    })
})
