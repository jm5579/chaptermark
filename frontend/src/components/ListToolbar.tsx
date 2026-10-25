import {SORT_LABELS, SORT_MODES, type SortMode} from '../lib/view'

export interface ListFilters {
    sort: SortMode
    staleOnly: boolean
    tag: string
}

/** Sort menu, "not read in 6+ months" toggle and tag buttons above the list. */
export function ListToolbar({value, tagCounts, onChange}: {
    value: ListFilters
    tagCounts: Record<string, number>
    onChange: (f: ListFilters) => void
}) {
    const tags = Object.keys(tagCounts).sort((a, b) => a.localeCompare(b, undefined, {sensitivity: 'base'}))
    const activeMissing = value.tag && !tags.some(t => t.toLowerCase() === value.tag.toLowerCase())
    return (
        <div className="list-toolbar">
            <label className="sort">
                <span className="muted small">Sort</span>
                <select value={value.sort} onChange={e => onChange({...value, sort: e.target.value as SortMode})}>
                    {SORT_MODES.map(m => <option key={m} value={m}>{SORT_LABELS[m]}</option>)}
                </select>
            </label>
            <label className="toggle">
                <input type="checkbox" checked={value.staleOnly}
                       onChange={e => onChange({...value, staleOnly: e.target.checked})}/>
                Not read in 6+ months
            </label>
            {(tags.length > 0 || value.tag) && (
                <div className="tag-filter" role="group" aria-label="Filter by tag">
                    <button className={!value.tag ? 'on' : ''} onClick={() => onChange({...value, tag: ''})}>All</button>
                    {tags.map(t => (
                        <button key={t} className={value.tag.toLowerCase() === t.toLowerCase() ? 'on' : ''}
                                onClick={() => onChange({...value, tag: value.tag.toLowerCase() === t.toLowerCase() ? '' : t})}>
                            {t} <span className="count">{tagCounts[t]}</span>
                        </button>
                    ))}
                    {activeMissing && <button className="on" onClick={() => onChange({...value, tag: ''})}>{value.tag} <span className="count">0</span></button>}
                </div>
            )}
        </div>
    )
}
