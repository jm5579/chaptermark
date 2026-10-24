import {useEffect, useRef, useState} from 'react'
import {api, errorMessage, type ItemView, type OnlineResult} from '../lib/api'
import {useDebounced, useEscape} from '../lib/hooks'
import {formatChapter} from '../lib/time'
import {STATUS_LABELS, TYPE_LABELS, type StatusKey, type TypeKey} from '../lib/view'
import {Cover} from './Cover'

function typeLabel(r: OnlineResult): string {
    return r.mappedType in TYPE_LABELS ? TYPE_LABELS[r.mappedType as TypeKey] : r.type || 'Other'
}

/** The top-bar search box with its results dropdown. */
export function SearchPanel({onOpen, onAdd}: {
    onOpen: (seriesId: number) => void
    onAdd: (r: OnlineResult) => void
}) {
    const [q, setQ] = useState('')
    const [open, setOpen] = useState(false)
    const [showOther, setShowOther] = useState(false)
    const [local, setLocal] = useState<ItemView[]>([])
    const [online, setOnline] = useState<OnlineResult[] | null>(null)
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')
    const box = useRef<HTMLDivElement>(null)
    const debounced = useDebounced(q.trim(), 400)

    const close = () => setOpen(false)
    useEscape(open, close)

    useEffect(() => {
        const onDown = (e: MouseEvent) => {
            if (box.current && !box.current.contains(e.target as Node)) close()
        }
        window.addEventListener('mousedown', onDown)
        return () => window.removeEventListener('mousedown', onDown)
    }, [])

    useEffect(() => {
        let stale = false
        const query = q.trim()
        if (!query) {
            setLocal([])
            return
        }
        api.SearchLocal(query).then(r => !stale && setLocal(r ?? [])).catch(() => undefined)
        return () => {
            stale = true
        }
    }, [q])

    useEffect(() => {
        let stale = false
        setError('')
        if (debounced.length < 2) {
            setOnline(null)
            return
        }
        setLoading(true)
        api.SearchOnline(debounced, 1, showOther)
            .then(r => !stale && setOnline(r ?? []))
            .catch(e => !stale && setError(errorMessage(e)))
            .finally(() => !stale && setLoading(false))
        return () => {
            stale = true
        }
    }, [debounced, showOther])

    const pick = (fn: () => void) => {
        fn()
        close()
    }

    return (
        <div className="search" ref={box}>
            <input className="search-input" placeholder="Search your list or add a new series..." value={q}
                   onChange={e => {
                       setQ(e.target.value)
                       setOpen(true)
                   }}
                   onFocus={() => setOpen(true)} spellCheck={false}/>
            {open && q.trim() && (
                <div className="search-panel">
                    <div className="search-section">
                        <h3>In your list</h3>
                        {local.length === 0
                            ? <p className="faint small">No match in your list.</p>
                            : local.map(it => (
                                <button key={it.series.id} className="result" onClick={() => pick(() => onOpen(it.series.id))}>
                                    <Cover file={it.series.coverFile} title={it.series.title} seed={it.series.id} className="thumb"/>
                                    <span className="result-main">
                                        <span className="t">{it.series.title}</span>
                                        <span className="faint small">
                                            {STATUS_LABELS[it.entry.status as StatusKey]} · ch {formatChapter(it.entry.myChapter) === '-' ? 0 : formatChapter(it.entry.myChapter)} of {formatChapter(it.effectiveLatest)}
                                        </span>
                                    </span>
                                    {it.newCount > 0 && it.entry.status !== 'plan' && <span className="badge-new">+{it.newCount}</span>}
                                </button>
                            ))}
                    </div>
                    <div className="search-section">
                        <h3>
                            Add from MangaUpdates
                            <label className="other-toggle">
                                <input type="checkbox" checked={showOther} onChange={e => setShowOther(e.target.checked)}/>
                                show other types
                            </label>
                        </h3>
                        {error && <p className="error-text small">{error}</p>}
                        {!error && debounced.length < 2 && <p className="faint small">Keep typing...</p>}
                        {!error && loading && <p className="faint small">Searching MangaUpdates...</p>}
                        {!error && !loading && online?.length === 0 && <p className="faint small">No results on MangaUpdates.</p>}
                        {!loading && online?.map(r => (
                            <div key={r.id} className="result">
                                <Cover url={r.coverUrl} title={r.title} seed={r.id} className="thumb"/>
                                <span className="result-main">
                                    <span className="t">{r.title}</span>
                                    {r.hitTitle && r.hitTitle !== r.title &&
                                        <span className="faint small">matched: {r.hitTitle}</span>}
                                    <span className="small result-meta">
                                        <span className="pill">{typeLabel(r)}</span>
                                        {r.year && <span className="faint">{r.year}</span>}
                                        <span className="muted">MU {r.rating > 0 ? r.rating.toFixed(1) : '-'}</span>
                                    </span>
                                </span>
                                {r.inLibrary
                                    ? <button className="chip" onClick={() => pick(() => onOpen(r.seriesId))}>In your list</button>
                                    : <button className="primary" onClick={() => pick(() => onAdd(r))}>Add</button>}
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    )
}
