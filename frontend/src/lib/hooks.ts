import {useEffect, useState} from 'react'

/** Current time, refreshed every intervalMs so relative times stay fresh. */
export function useNow(intervalMs = 60_000): Date {
    const [now, setNow] = useState(() => new Date())
    useEffect(() => {
        const id = setInterval(() => setNow(new Date()), intervalMs)
        return () => clearInterval(id)
    }, [intervalMs])
    return now
}

/** value, but only after it has stopped changing for delayMs. */
export function useDebounced<T>(value: T, delayMs: number): T {
    const [debounced, setDebounced] = useState(value)
    useEffect(() => {
        const id = setTimeout(() => setDebounced(value), delayMs)
        return () => clearTimeout(id)
    }, [value, delayMs])
    return debounced
}

/** Calls onEscape when the Escape key is pressed while active. */
export function useEscape(active: boolean, onEscape: () => void) {
    useEffect(() => {
        if (!active) return
        const h = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onEscape()
        }
        window.addEventListener('keydown', h)
        return () => window.removeEventListener('keydown', h)
    }, [active, onEscape])
}
