import {describe, expect, it} from 'vitest'
import {rhythmLabel, seriesBadge} from './rhythm'

const now = new Date('2026-10-24T12:00:00Z')

describe('rhythmLabel', () => {
    it.each([
        [0, ''],
        [2, 'every few days'],
        [7, 'weekly'],
        [7.5, 'weekly'],
        [14, 'every 2 weeks'],
        [30, 'monthly'],
        [90, 'irregular'],
    ])('%d days -> %j', (days, want) => {
        expect(rhythmLabel(days)).toBe(want)
    })
})

describe('seriesBadge', () => {
    const s = (o: Partial<{pubStatus: string, completed: boolean, releaseIntervalDays: number, lastReleaseDate: string}>) => ({
        pubStatus: '', completed: false, releaseIntervalDays: 0, lastReleaseDate: '', ...o,
    })
    it('marks a hiatus from the MangaUpdates status text', () => {
        expect(seriesBadge(s({pubStatus: '311 Chapters + Prologue (Hiatus)'}), now)).toBe('hiatus')
    })
    it('marks a finished series', () => {
        expect(seriesBadge(s({completed: true}), now)).toBe('finished')
    })
    it('marks a series quiet after about 3 of its usual gaps without a release', () => {
        expect(seriesBadge(s({releaseIntervalDays: 7, lastReleaseDate: '2026-09-20'}), now)).toBe('quiet')
        expect(seriesBadge(s({releaseIntervalDays: 7, lastReleaseDate: '2026-10-14'}), now)).toBe(null)
    })
    it('waits at least 2 weeks before calling a fast series quiet', () => {
        expect(seriesBadge(s({releaseIntervalDays: 2, lastReleaseDate: '2026-10-15'}), now)).toBe(null)
    })
    it('marks quiet after 90 days when the rhythm is unknown', () => {
        expect(seriesBadge(s({lastReleaseDate: '2026-06-01'}), now)).toBe('quiet')
        expect(seriesBadge(s({}), now)).toBe(null)
    })
    it('prefers hiatus over finished over quiet', () => {
        expect(seriesBadge(s({pubStatus: 'Hiatus', completed: true, lastReleaseDate: '2020-01-01'}), now)).toBe('hiatus')
    })
})
