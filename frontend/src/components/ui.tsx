import type {ReactNode} from 'react'
import {useEscape} from '../lib/hooks'

/** A centred dialog over a dimmed backdrop. Esc or a backdrop click closes it. */
export function Modal({title, onClose, children, wide = false}: {
    title: string
    onClose: () => void
    children: ReactNode
    wide?: boolean
}) {
    useEscape(true, onClose)
    return (
        <div className="backdrop" onMouseDown={e => e.target === e.currentTarget && onClose()}>
            <div className={`modal ${wide ? 'wide' : ''}`} role="dialog" aria-modal="true" aria-label={title}>
                <div className="modal-head">
                    <h2>{title}</h2>
                    <button className="ghost icon-btn" onClick={onClose} aria-label="Close">✕</button>
                </div>
                {children}
            </div>
        </div>
    )
}

/** Owner rating 1-10; clicking the selected value clears it (unrated). */
export function RatingInput({value, onChange}: { value: number; onChange: (v: number) => void }) {
    return (
        <div className="rating-input" role="radiogroup" aria-label="Your rating">
            {Array.from({length: 10}, (_, i) => i + 1).map(n => (
                <button key={n} type="button" role="radio" aria-checked={n === value}
                        className={n <= value ? 'on' : ''} onClick={() => onChange(n === value ? 0 : n)}
                        title={n === value ? 'Clear rating' : `${n}/10`}>
                    {n}
                </button>
            ))}
            <span className="rating-label">{value > 0 ? `★ ${value}/10` : 'Unrated'}</span>
        </div>
    )
}
