import {describe, expect, it} from 'vitest'
import {compareItems, hasNew, selectView, type ItemLike} from './view'

let nextId = 1
function item(type: string, status: string, newCount: number, myRating: number, lastReadAt = '0001-01-01T00:00:00Z'): ItemLike & {id: number} {
    return {id: nextId++, effectiveType: type, newCount, entry: {status, myRating, lastReadAt}}
}

describe('hasNew', () => {
    it('only counts reading and on-hold series', () => {
        expect(hasNew(item('manhwa', 'reading', 3, 0))).toBe(true)
        expect(hasNew(item('manhwa', 'on_hold', 3, 0))).toBe(true)
        expect(hasNew(item('manhwa', 'plan', 311, 0))).toBe(false)
        expect(hasNew(item('manhwa', 'completed', 3, 0))).toBe(false)
        expect(hasNew(item('manhwa', 'reading', 0, 0))).toBe(false)
    })
})

describe('selectView', () => {
    const orv = item('manhwa', 'reading', 36, 10)
    const hua = item('manhwa', 'on_hold', 33, 9)
    const nano = item('manhwa', 'reading', 21, 9, '2026-09-01T00:00:00Z')
    const unrated = item('manhwa', 'reading', 4, 0)
    const caughtUp = item('manhwa', 'reading', 0, 10)
    const planned = item('manhwa', 'plan', 311, 8)
    const kingdom = item('manga', 'reading', 22, 8)
    const martial = item('manhua', 'reading', 104, 7)
    const done = item('manga', 'completed', 0, 9)
    const all = [planned, unrated, caughtUp, martial, kingdom, nano, hua, orv, done]

    it('puts only new reading/on-hold series of the selected type on the shelf, by rating', () => {
        const v = selectView(all, 'manhwa', 'reading')
        expect(v.shelf.map(i => i.id)).toEqual([orv.id, nano.id, hua.id, unrated.id])
    })

    it('never puts plan-to-read series on the shelf', () => {
        const v = selectView(all, 'manhwa', 'plan')
        expect(v.shelf).not.toContain(planned)
        expect(v.list).toEqual([planned])
    })

    it('lists the tab with new chapters first, then rating, unrated last', () => {
        const v = selectView(all, 'manhwa', 'reading')
        expect(v.list.map(i => i.id)).toEqual([orv.id, nano.id, unrated.id, caughtUp.id])
    })

    it('counts new series per type regardless of the selected type', () => {
        const v = selectView(all, 'manga', 'completed')
        expect(v.newCounts).toEqual({manhwa: 4, manga: 1, manhua: 1})
        expect(v.list).toEqual([done])
    })

    it('counts tabs within the selected type', () => {
        const v = selectView(all, 'manhwa', 'reading')
        expect(v.tabCounts).toEqual({reading: 4, plan: 1, completed: 0, on_hold: 1, dropped: 0})
    })
})

describe('compareItems', () => {
    it('breaks rating ties by most recently read', () => {
        const older = item('manga', 'reading', 0, 7, '2026-01-01T00:00:00Z')
        const newer = item('manga', 'reading', 0, 7, '2026-06-01T00:00:00Z')
        expect([older, newer].sort(compareItems)).toEqual([newer, older])
    })
})
