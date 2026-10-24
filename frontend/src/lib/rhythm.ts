// How often a series gets new chapters, and whether it has gone quiet.

const DAY_MS = 24 * 60 * 60 * 1000

/** "weekly", "every 2 weeks"... from the typical days between new chapters ('' if unknown). */
export function rhythmLabel(days: number): string {
    if (!days || days <= 0) return ''
    if (days <= 4) return 'every few days'
    if (days <= 10) return 'weekly'
    if (days <= 20) return 'every 2 weeks'
    if (days <= 45) return 'monthly'
    return 'irregular'
}

export type Badge = 'hiatus' | 'finished' | 'quiet'

export const BADGE_LABELS: Record<Badge, string> = {hiatus: 'Hiatus', finished: 'Finished', quiet: 'Quiet'}

export const BADGE_HINTS: Record<Badge, string> = {
    hiatus: 'MangaUpdates lists this series as on hiatus',
    finished: 'This series has finished and is fully translated',
    quiet: 'No new chapter for much longer than usual',
}

interface RhythmFields {
    pubStatus: string
    completed: boolean
    releaseIntervalDays: number
    lastReleaseDate: string
}

/**
 * Hiatus (from MangaUpdates' status text), Finished (fully released), or Quiet
 * (no release for about 3 of its usual gaps, at least 2 weeks; 90 days when
 * the rhythm is unknown). Null when none apply.
 */
export function seriesBadge(s: RhythmFields, now: Date): Badge | null {
    if (/hiatus/i.test(s.pubStatus ?? '')) return 'hiatus'
    if (s.completed) return 'finished'
    if (!s.lastReleaseDate) return null
    const since = (now.getTime() - new Date(s.lastReleaseDate + 'T00:00:00Z').getTime()) / DAY_MS
    if (Number.isNaN(since)) return null
    const limit = s.releaseIntervalDays > 0 ? Math.max(3 * s.releaseIntervalDays, 14) : 90
    return since > limit ? 'quiet' : null
}
