// Small display helpers for times and chapter numbers.

/** Go's zero time.Time serialises as 0001-01-01T00:00:00Z. */
export function isZeroTime(iso: any): boolean {
    return !iso || String(iso).startsWith('0001-01-01')
}

function plural(n: number, unit: string): string {
    return `${n} ${unit}${n === 1 ? '' : 's'} ago`
}

/** "just now", "5 min ago", "3 h ago", "4 days ago", "2 weeks ago", "3 months ago", "1 year ago" or "never". */
export function relativeTime(iso: any, now: Date = new Date()): string {
    if (isZeroTime(iso)) return 'never'
    const then = new Date(iso).getTime()
    if (Number.isNaN(then)) return 'never'
    const mins = Math.max(0, Math.floor((now.getTime() - then) / 60000))
    if (mins < 1) return 'just now'
    if (mins < 60) return `${mins} min ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours} h ago`
    const days = Math.floor(hours / 24)
    if (days < 14) return plural(days, 'day')
    if (days < 56) return plural(Math.floor(days / 7), 'week')
    if (days < 365) return plural(Math.floor(days / 30), 'month')
    return plural(Math.floor(days / 365), 'year')
}

/** 311 -> "311", 10.5 -> "10.5", 0 -> "-" (unknown). */
export function formatChapter(n: number): string {
    if (!n) return '-'
    return Number.isInteger(n) ? String(n) : String(Math.round(n * 10) / 10)
}

/**
 * The chapter the +1 / -1 buttons should save: steps from what's typed in the box
 * (if it's a valid number) so a just-typed value isn't lost, else from the
 * saved chapter. Never below 0.
 */
export function nextChapter(input: string, saved: number, delta: number): number {
    const typed = input.trim() === '' ? NaN : Number(input)
    const base = Number.isFinite(typed) && typed >= 0 ? typed : saved
    return Math.max(0, Math.floor(base) + delta)
}

/** "2026-05-19" -> "19 May 2026"; anything else is returned unchanged. */
export function formatDate(ymd: string): string {
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(ymd ?? '')
    if (!m) return ymd ?? ''
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
    return `${Number(m[3])} ${months[Number(m[2]) - 1]} ${m[1]}`
}
