import {useEffect, useState} from 'react'
import {api, errorMessage, type Tag} from '../lib/api'

/** Rename or delete tags (Settings). */
export function TagManager({onChanged}: { onChanged: () => void }) {
    const [tags, setTags] = useState<Tag[]>([])
    const [editing, setEditing] = useState<number | null>(null)
    const [name, setName] = useState('')
    const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
    const [error, setError] = useState('')

    const load = () => api.ListTags().then(t => setTags(t ?? [])).catch(e => setError(errorMessage(e)))
    useEffect(() => {
        load()
    }, [])

    const run = async (fn: () => Promise<void>) => {
        setError('')
        try {
            await fn()
            setEditing(null)
            setConfirmDelete(null)
            await load()
            onChanged()
        } catch (e) {
            setError(errorMessage(e))
        }
    }

    return (
        <section className="panel-section">
            <h3>Tags</h3>
            {tags.length === 0
                ? <p className="faint small">No tags yet. Add them from a series panel.</p>
                : <ul className="tag-list">
                    {tags.map(t => (
                        <li key={t.id}>
                            {editing === t.id
                                ? <form className="tag-rename" onSubmit={e => {
                                    e.preventDefault()
                                    run(() => api.RenameTag(t.id, name))
                                }}>
                                    <input value={name} maxLength={40} autoFocus aria-label="New tag name"
                                           onChange={e => setName(e.target.value)}/>
                                    <button type="submit" className="primary">Save</button>
                                    <button type="button" className="ghost" onClick={() => setEditing(null)}>Cancel</button>
                                </form>
                                : <>
                                    <span className="tag-name">{t.name}</span>
                                    <span className="faint small">{t.count} series</span>
                                    {confirmDelete === t.id
                                        ? <>
                                            <span className="small">Remove from all series?</span>
                                            <button className="danger" onClick={() => run(() => api.DeleteTag(t.id))}>Delete</button>
                                            <button className="ghost" onClick={() => setConfirmDelete(null)}>Cancel</button>
                                        </>
                                        : <>
                                            <button className="ghost" onClick={() => {
                                                setEditing(t.id)
                                                setName(t.name)
                                            }}>Rename</button>
                                            <button className="ghost" onClick={() => setConfirmDelete(t.id)}>Delete</button>
                                        </>}
                                </>}
                        </li>
                    ))}
                </ul>}
            {error && <p className="error-text small">{error}</p>}
        </section>
    )
}
