import {useState} from 'react'
import {api, errorMessage, type AddInput, type ItemView, type OnlineResult} from '../lib/api'
import {STATUS_LABELS, STATUSES, TYPE_LABELS, TYPES, type StatusKey, type TypeKey} from '../lib/view'
import {Cover} from './Cover'
import {Modal, RatingInput} from './ui'

/** Confirms status / chapter / rating before adding a MangaUpdates series. */
export function AddDialog({result, onClose, onAdded}: {
    result: OnlineResult
    onClose: () => void
    onAdded: (v: ItemView) => void
}) {
    const needsType = result.mappedType === 'other'
    const [status, setStatus] = useState<StatusKey>('reading')
    const [chapter, setChapter] = useState('')
    const [rating, setRating] = useState(0)
    const [type, setType] = useState<TypeKey>('manga')
    const [busy, setBusy] = useState(false)
    const [error, setError] = useState('')

    const submit = async () => {
        const myChapter = chapter.trim() === '' ? 0 : Number(chapter)
        if (!Number.isFinite(myChapter) || myChapter < 0) {
            setError('Chapter must be a number (0 or more).')
            return
        }
        setBusy(true)
        setError('')
        try {
            const input = {status, myChapter, myRating: rating, typeOverride: needsType ? type : ''} as AddInput
            onAdded(await api.AddSeries(result.id, input))
        } catch (e) {
            setError(errorMessage(e))
            setBusy(false)
        }
    }

    return (
        <Modal title="Add to your list" onClose={busy ? () => undefined : onClose}>
            <div className="add-head">
                <Cover url={result.coverUrl} title={result.title} seed={result.id} className="add-cover"/>
                <div>
                    <div className="add-title">{result.title}</div>
                    <div className="faint small">{result.type}{result.year ? ` · ${result.year}` : ''}
                        {result.rating > 0 ? ` · MU ${result.rating.toFixed(1)}` : ''}</div>
                </div>
            </div>
            <form className="form" onSubmit={e => {
                e.preventDefault()
                submit()
            }}>
                <label>Status
                    <select value={status} onChange={e => setStatus(e.target.value as StatusKey)}>
                        {STATUSES.map(s => <option key={s} value={s}>{STATUS_LABELS[s]}</option>)}
                    </select>
                </label>
                <label>Last chapter you read
                    <input type="number" min={0} step="any" placeholder="0" value={chapter}
                           onChange={e => setChapter(e.target.value)} autoFocus/>
                </label>
                {needsType && (
                    <label>Show it under
                        <select value={type} onChange={e => setType(e.target.value as TypeKey)}>
                            {TYPES.map(t => <option key={t} value={t}>{TYPE_LABELS[t]}</option>)}
                        </select>
                        <span className="faint small">MangaUpdates lists this as “{result.type}”.</span>
                    </label>
                )}
                <div className="field">Your rating <RatingInput value={rating} onChange={setRating}/></div>
                {error && <p className="error-text">{error}</p>}
                <div className="actions">
                    <button type="button" className="ghost" onClick={onClose} disabled={busy}>Cancel</button>
                    <button type="submit" className="primary" disabled={busy}>
                        {busy ? 'Adding… fetching details' : 'Add series'}
                    </button>
                </div>
            </form>
        </Modal>
    )
}
