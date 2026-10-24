import {describe, expect, it} from 'vitest'
import {formatChapter, isZeroTime, relativeTime} from './time'

const now = new Date('2026-10-23T12:00:00Z')

describe('relativeTime', () => {
    it.each([
        ['2026-10-23T11:59:30Z', 'just now'],
        ['2026-10-23T11:55:00Z', '5 min ago'],
        ['2026-10-23T09:00:00Z', '3 h ago'],
        ['2026-10-22T12:00:00Z', '1 day ago'],
        ['2026-10-19T12:00:00Z', '4 days ago'],
        ['2026-10-09T12:00:00Z', '2 weeks ago'],
        ['2026-07-23T12:00:00Z', '3 months ago'],
        ['2025-10-01T12:00:00Z', '1 year ago'],
        ['2023-01-01T12:00:00Z', '3 years ago'],
        ['0001-01-01T00:00:00Z', 'never'],
        ['', 'never'],
    ])('%s → %s', (iso, want) => {
        expect(relativeTime(iso, now)).toBe(want)
    })
})

describe('isZeroTime', () => {
    it('treats Go zero time and empty as zero', () => {
        expect(isZeroTime('0001-01-01T00:00:00Z')).toBe(true)
        expect(isZeroTime('')).toBe(true)
        expect(isZeroTime('2026-10-23T12:00:00Z')).toBe(false)
    })
})

describe('formatChapter', () => {
    it.each([
        [311, '311'],
        [10.5, '10.5'],
        [0, '–'],
    ])('%s → %s', (n, want) => {
        expect(formatChapter(n)).toBe(want)
    })
})
