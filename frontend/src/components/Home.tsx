import type {ItemView} from '../lib/api'
import {formatChapter, relativeTime} from '../lib/time'
import {
    hasNew, pickAltTitles, STATUS_LABELS, STATUSES, TYPE_LABELS, TYPES,
    type StatusKey, type TypeKey,
} from '../lib/view'
import {Cover} from './Cover'

export function TypeSwitch({value, newCounts, onChange}: {
    value: TypeKey
    newCounts: Record<TypeKey, number>
    onChange: (t: TypeKey) => void
}) {
    return (
        <div className="type-switch" role="tablist">
            {TYPES.map(t => (
                <button key={t} role="tab" aria-selected={t === value}
                        className={t === value ? 'on' : ''} onClick={() => onChange(t)}>
                    {TYPE_LABELS[t]}
                    {newCounts[t] > 0 && <em>{newCounts[t]} new</em>}
                </button>
            ))}
        </div>
    )
}

export function Shelf({items, onOpen}: { items: ItemView[]; onOpen: (id: number) => void }) {
    if (items.length === 0) return null
    return (
        <section className="shelf-section">
            <h2 className="section-title"><b>New chapters · {items.length}</b><span>sorted by your rating</span></h2>
            <div className="shelf">
                {items.map(it => (
                    <button key={it.series.id} className="shelf-card" onClick={() => onOpen(it.series.id)}
                            title={it.series.title}>
                        <Cover file={it.series.coverFile} title={it.series.title} seed={it.series.id}>
                            <span className="badge-new cover-badge">+{it.newCount}</span>
                        </Cover>
                        <span className="shelf-title">{it.series.title}</span>
                        <span className="shelf-meta">
                            {formatChapter(it.entry.myChapter)} → {formatChapter(it.effectiveLatest)}
                            {it.entry.myRating > 0 && <span className="star"> · ★{it.entry.myRating}</span>}
                        </span>
                    </button>
                ))}
            </div>
        </section>
    )
}

export function StatusTabs({value, counts, onChange}: {
    value: StatusKey
    counts: Record<StatusKey, number>
    onChange: (s: StatusKey) => void
}) {
    return (
        <div className="status-tabs" role="tablist">
            {STATUSES.map(s => (
                <button key={s} role="tab" aria-selected={s === value}
                        className={s === value ? 'on' : ''} onClick={() => onChange(s)}>
                    {STATUS_LABELS[s]} · {counts[s]}
                </button>
            ))}
        </div>
    )
}

function altLine(it: ItemView): string {
    return pickAltTitles(it.series.title, it.series.altTitles, it.effectiveType,
        {romanized: it.series.romanizedTitle, native: it.series.nativeTitle}).join(' · ')
}

export function SeriesRow({it, now, onOpen}: { it: ItemView; now: Date; onOpen: (id: number) => void }) {
    const latest = it.effectiveLatest
    const mine = it.entry.myChapter
    const pct = latest > 0 ? Math.min(100, (mine / latest) * 100) : 0
    const isPlan = it.entry.status === 'plan'
    return (
        <button className="series-row" onClick={() => onOpen(it.series.id)} aria-label={it.series.title}>
            <Cover file={it.series.coverFile} title={it.series.title} seed={it.series.id} className="thumb"/>
            <span className="row-title">
                <span className="t">{it.series.title}</span>
                <span className="a">{altLine(it)}</span>
            </span>
            <span className="row-progress">
                {isPlan
                    ? <span className="muted">{latest > 0 ? `${formatChapter(latest)} ch` : '–'}</span>
                    : <>
                        <span>{formatChapter(mine) === '–' ? '0' : formatChapter(mine)} / {formatChapter(latest)}</span>
                        <span className="bar"><i style={{width: `${pct}%`}}/></span>
                    </>}
            </span>
            <span>{hasNew(it) && <span className="badge-new">+{it.newCount}</span>}</span>
            <span className={it.entry.myRating > 0 ? 'star' : 'faint'}>
                {it.entry.myRating > 0 ? `★ ${it.entry.myRating}` : '★ –'}
            </span>
            <span className="muted">MU {it.series.muRating > 0 ? it.series.muRating.toFixed(1) : '–'}</span>
            <span className="faint when">{isPlan ? '' : `read ${relativeTime(it.entry.lastReadAt, now)}`}</span>
        </button>
    )
}

export function EmptyState({text}: { text: string }) {
    return <div className="empty">{text}</div>
}
