import type {ReactNode} from 'react'
import type {Progress} from '../lib/api'
import {isZeroTime, relativeTime} from '../lib/time'

function checkLabel(p: Progress | null, now: Date): string {
    if (!p) return '↻ Check for new chapters'
    if (p.running) return p.total > 0 ? `Checking ${p.done}/${p.total}…` : 'Checking…'
    const last = isZeroTime(p.lastCheckedAt) ? '' : relativeTime(p.lastCheckedAt, now)
    if (p.error === 'offline') return last ? `Couldn't check — offline · last ${last}` : `Couldn't check — offline`
    return last ? `↻ Checked ${last}` : '↻ Check for new chapters'
}

export function TopBar({search, progress, now, onCheck, onSettings}: {
    search: ReactNode
    progress: Progress | null
    now: Date
    onCheck: () => void
    onSettings: () => void
}) {
    const label = checkLabel(progress, now)
    return (
        <header className="topbar">
            <span className="logo">CHAPTERMARK</span>
            <div className="search-slot">{search}</div>
            <button className={`check-btn ${progress?.error === 'offline' ? 'offline' : ''}`}
                    onClick={onCheck} disabled={!!progress?.running}
                    title={progress?.error === 'partial' ? `${progress.failed} series couldn't be checked` : 'Check for new chapters'}>
                {label}
            </button>
            <button className="ghost icon-btn" onClick={onSettings} aria-label="Settings" title="Settings">⚙</button>
        </header>
    )
}
