import {useEffect, useState} from 'react'
import {api, errorMessage, toUpdate, type EntryUpdate, type ItemView} from '../lib/api'
import {useEscape} from '../lib/hooks'
import {formatChapter, formatDate, nextChapter, relativeTime} from '../lib/time'
import {STATUS_LABELS, STATUSES, TYPE_LABELS, TYPES, type StatusKey, type TypeKey} from '../lib/view'
import {Cover} from './Cover'
import {RatingInput} from './ui'
import {TagEditor} from './TagEditor'
import {BADGE_HINTS, BADGE_LABELS, rhythmLabel, seriesBadge} from '../lib/rhythm'

function muTypeLabel(t: string): string {
    return t in TYPE_LABELS ? TYPE_LABELS[t as TypeKey] : 'Other'
}

/** Right-hand panel for one series: update progress, rating, status, notes. */
export function DetailPanel({item, now, allTags, onClose, onChange, onRemoved}: {
    item: ItemView
    now: Date
    allTags: string[]
    onClose: () => void
    onChange: (v: ItemView) => void
    onRemoved: (id: number) => void
}) {
    const s = item.series
    const e = item.entry
    const [chapter, setChapter] = useState(String(e.myChapter))
    const [notes, setNotes] = useState(e.notes)
    const [readUrl, setReadUrl] = useState(e.readUrl)
    const [manual, setManual] = useState(s.latestManual != null ? String(s.latestManual) : '')
    const [confirmRemove, setConfirmRemove] = useState(false)
    const [error, setError] = useState('')
    useEscape(true, onClose)

    // Reset local fields when another series is opened or the item changes underneath.
    useEffect(() => setChapter(String(e.myChapter)), [s.id, e.myChapter])
    useEffect(() => setNotes(e.notes), [s.id, e.notes])
    useEffect(() => setReadUrl(e.readUrl), [s.id, e.readUrl])
    useEffect(() => setManual(s.latestManual != null ? String(s.latestManual) : ''), [s.id, s.latestManual])
    useEffect(() => setConfirmRemove(false), [s.id])

    const save = async (patch: Partial<EntryUpdate>) => {
        setError('')
        try {
            onChange(await api.UpdateEntry(s.id, toUpdate(item, patch)))
        } catch (err) {
            setError(errorMessage(err))
        }
    }

    const commitChapter = (value: string) => {
        const n = Number(value)
        if (value.trim() === '' || !Number.isFinite(n) || n < 0) {
            setChapter(String(e.myChapter))
            return
        }
        if (n !== e.myChapter) save({myChapter: n})
    }

    const step = (delta: number) => {
        const n = nextChapter(chapter, e.myChapter, delta)
        setChapter(String(n))
        save({myChapter: n})
    }

    const commitManual = (value: string) => {
        if (value.trim() === '') {
            if (s.latestManual != null) save({latestManual: undefined})
            return
        }
        const n = Number(value)
        if (!Number.isFinite(n) || n < 0) {
            setManual(s.latestManual != null ? String(s.latestManual) : '')
            return
        }
        if (n !== s.latestManual) save({latestManual: n})
    }

    const remove = async () => {
        try {
            await api.RemoveSeries(s.id)
            onRemoved(s.id)
        } catch (err) {
            setError(errorMessage(err))
        }
    }

    const otherType = s.type === 'other' || !s.type
    const badge = seriesBadge(s, now)
    const rhythm = rhythmLabel(s.releaseIntervalDays)
    const rhythmLine = [
        rhythm && `Usually ${rhythm === 'irregular' ? 'irregular' : rhythm}`,
        s.lastReleaseDate && `last release ${relativeTime(s.lastReleaseDate + 'T00:00:00Z', now)}`,
    ].filter(Boolean).join(' · ')
    const latestLine = [
        s.latestRaw ? `Ch ${s.latestRaw}` : (s.latestChapter > 0 ? `Ch ${formatChapter(s.latestChapter)}` : 'No releases found yet'),
        s.latestGroup,
        s.latestDate ? formatDate(s.latestDate) : '',
    ].filter(Boolean).join(' · ')

    return (
        <div className="panel-backdrop" onMouseDown={ev => ev.target === ev.currentTarget && onClose()}>
            <aside className="panel" aria-label={s.title}>
                <button className="ghost icon-btn panel-close" onClick={onClose} aria-label="Close">✕</button>

                <div className="panel-head">
                    <Cover file={s.coverFile} url={s.coverUrl} title={s.title} seed={s.id} className="panel-cover">
                        {item.newCount > 0 && e.status !== 'plan' && <span className="badge-new cover-badge">+{item.newCount}</span>}
                    </Cover>
                    <div className="panel-head-text">
                        <h2>{s.title}</h2>
                        {badge && <span className={`status-badge ${badge}`} title={BADGE_HINTS[badge]}>{BADGE_LABELS[badge]}</span>}
                        <div className="faint small">{[s.year, s.pubStatus.split('\n')[0]].filter(Boolean).join(' · ')}</div>
                        <label className="inline-field">Type
                            <select value={s.typeOverride || ''} onChange={ev => save({typeOverride: ev.target.value})}>
                                {!otherType && <option value="">{muTypeLabel(s.type)} (from MangaUpdates)</option>}
                                {TYPES.map(t => <option key={t} value={t}>{TYPE_LABELS[t]}</option>)}
                            </select>
                        </label>
                    </div>
                </div>

                <section className="panel-section">
                    <h3>Your progress</h3>
                    <div className="chapter-stepper">
                        <button onClick={() => step(-1)} disabled={e.myChapter <= 0} aria-label="One chapter back">-1</button>
                        <input type="number" min={0} step="any" value={chapter} aria-label="Last chapter read"
                               onChange={ev => setChapter(ev.target.value)}
                               onBlur={ev => commitChapter(ev.target.value)}
                               onKeyDown={ev => ev.key === 'Enter' && (ev.target as HTMLInputElement).blur()}/>
                        <button className="primary" onClick={() => step(1)} aria-label="One chapter forward">+1</button>
                        <span className="muted">of {formatChapter(item.effectiveLatest)}</span>
                    </div>
                    <div className="faint small">Last read {relativeTime(e.lastReadAt, now)}</div>
                </section>

                <section className="panel-section">
                    <h3>Latest translated chapter</h3>
                    <div>{latestLine}</div>
                    {rhythmLine && <div className="muted small">{rhythmLine}</div>}
                    <label className="inline-field small">Set manually
                        <input type="number" min={0} step="any" placeholder="if the sources are behind" value={manual}
                               onChange={ev => setManual(ev.target.value)}
                               onBlur={ev => commitManual(ev.target.value)}
                               onKeyDown={ev => ev.key === 'Enter' && (ev.target as HTMLInputElement).blur()}/>
                        {s.latestManual != null &&
                            <button className="ghost small" onClick={() => save({latestManual: undefined})}>Clear</button>}
                    </label>
                </section>

                <section className="panel-section">
                    <h3>Status & rating</h3>
                    <select value={e.status} onChange={ev => save({status: ev.target.value})}>
                        {STATUSES.map(st => <option key={st} value={st}>{STATUS_LABELS[st as StatusKey]}</option>)}
                    </select>
                    <RatingInput value={e.myRating} onChange={v => save({myRating: v})}/>
                </section>

                <section className="panel-section">
                    <h3>Tags</h3>
                    <TagEditor tags={e.tags ?? []} suggestions={allTags} onChange={tags => save({tags})}/>
                </section>

                <section className="panel-section">
                    <h3>Where I read it</h3>
                    <div className="url-row">
                        <input type="url" placeholder="https://..." value={readUrl}
                               onChange={ev => setReadUrl(ev.target.value)}
                               onBlur={() => readUrl !== e.readUrl && save({readUrl: readUrl.trim()})}/>
                        <button onClick={() => api.OpenURL(e.readUrl)} disabled={!/^https?:\/\//.test(e.readUrl)}>Open</button>
                    </div>
                    <h3>Notes</h3>
                    <textarea rows={3} value={notes} placeholder="Anything to remember..."
                              onChange={ev => setNotes(ev.target.value)}
                              onBlur={() => notes !== e.notes && save({notes})}/>
                </section>

                <section className="panel-section">
                    <h3>Public ratings</h3>
                    <div className="ratings">
                        <span>MangaUpdates <b>{s.muRating > 0 ? s.muRating.toFixed(2) : '-'}</b>
                            {s.muVotes > 0 && <span className="faint"> ({s.muVotes.toLocaleString()} votes)</span>}</span>
                        <span>AniList <b>{s.alRating > 0 ? s.alRating.toFixed(1) : '-'}</b></span>
                    </div>
                    {s.muUrl && <button className="ghost link" onClick={() => api.OpenURL(s.muUrl)}>Open on MangaUpdates</button>}
                </section>

                {(s.altTitles?.length ?? 0) > 1 && (
                    <section className="panel-section">
                        <h3>Also known as</h3>
                        <div className="alt-titles">
                            {s.altTitles.filter(t => t !== s.title).map(t => <span key={t} className="pill">{t}</span>)}
                        </div>
                    </section>
                )}

                {error && <p className="error-text">{error}</p>}

                <section className="panel-section danger-zone">
                    {confirmRemove
                        ? <div className="confirm">
                            <span>Remove "{s.title}" from your list?</span>
                            <button className="ghost" onClick={() => setConfirmRemove(false)}>Cancel</button>
                            <button className="danger" onClick={remove}>Remove</button>
                        </div>
                        : <button className="danger" onClick={() => setConfirmRemove(true)}>Remove from list</button>}
                </section>
            </aside>
        </div>
    )
}
