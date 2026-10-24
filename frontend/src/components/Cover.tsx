import {useState, type ReactNode} from 'react'
import {coverSrc} from '../lib/api'

const gradients = [
    ['#3a5bd9', '#141a3a'], ['#c0392b', '#3a0f0b'], ['#16a085', '#0b2e28'], ['#d4a017', '#3b2a05'],
    ['#8e44ad', '#250f30'], ['#2980b9', '#0b2233'], ['#7f8c8d', '#1e2324'], ['#e67e22', '#3a1d05'],
]

function initials(title: string): string {
    const words = title.replace(/[^\p{L}\p{N} ]/gu, ' ').split(/\s+/).filter(Boolean)
    return words.slice(0, 3).map(w => w[0]).join('').toUpperCase() || '?'
}

interface Props {
    file?: string
    url?: string // remote URL (search results) when there is no cached file
    title: string
    seed: number
    className?: string
    children?: ReactNode
}

/** A cover image, or a coloured placeholder with initials when there is none. */
export function Cover({file, url, title, seed, className = '', children}: Props) {
    const [broken, setBroken] = useState(false)
    const src = coverSrc(file) || url || ''
    const [from, to] = gradients[Math.abs(seed) % gradients.length]
    return (
        <div className={`cover ${className}`} style={{background: `linear-gradient(160deg, ${from}, ${to})`}}>
            {src && !broken
                ? <img src={src} alt="" loading="lazy" draggable={false} onError={() => setBroken(true)}/>
                : <span className="cover-initials">{initials(title)}</span>}
            {children}
        </div>
    )
}
