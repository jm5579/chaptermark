import {useState} from 'react'

const MAX_TAGS = 20
const MAX_LENGTH = 40

/** Tag chips with remove buttons, and an input that suggests existing tags. */
export function TagEditor({tags, suggestions, onChange}: {
    tags: string[]
    suggestions: string[]
    onChange: (tags: string[]) => void
}) {
    const [draft, setDraft] = useState('')
    const has = (t: string) => tags.some(x => x.toLowerCase() === t.toLowerCase())
    const available = suggestions.filter(t => !has(t))

    const add = (raw: string) => {
        const name = raw.trim().slice(0, MAX_LENGTH)
        setDraft('')
        if (!name || has(name) || tags.length >= MAX_TAGS) return
        // Reuse the existing spelling when the tag already exists elsewhere.
        const known = suggestions.find(t => t.toLowerCase() === name.toLowerCase())
        onChange([...tags, known ?? name])
    }

    return (
        <div className="tag-editor">
            {tags.map(t => (
                <span key={t} className="tag-chip">
                    {t}
                    <button type="button" aria-label={`Remove tag ${t}`} title="Remove"
                            onClick={() => onChange(tags.filter(x => x !== t))}>✕</button>
                </span>
            ))}
            {tags.length < MAX_TAGS && (
                <input list="known-tags" value={draft} maxLength={MAX_LENGTH}
                       placeholder={tags.length ? 'Add another...' : 'Add a tag, e.g. Favourites'}
                       onChange={e => {
                           const v = e.target.value
                           if (v.endsWith(',')) add(v.slice(0, -1))
                           else setDraft(v)
                       }}
                       onKeyDown={e => {
                           if (e.key === 'Enter') {
                               e.preventDefault()
                               add(draft)
                           }
                       }}
                       onBlur={() => draft.trim() && add(draft)}/>
            )}
            <datalist id="known-tags">
                {available.map(t => <option key={t} value={t}/>)}
            </datalist>
        </div>
    )
}
