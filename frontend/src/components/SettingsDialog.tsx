import {useEffect, useState} from 'react'
import {api, errorMessage, type Settings} from '../lib/api'
import {Modal} from './ui'

type ImportStep = 'idle' | 'choose' | 'confirm-replace'

/** Settings, export/import and where the data lives. */
export function SettingsDialog({dataDir, version, onClose, onImported}: {
    dataDir: string
    version: string
    onClose: () => void
    onImported: () => void
}) {
    const [checkHours, setCheckHours] = useState('6')
    const [keep, setKeep] = useState('10')
    const [message, setMessage] = useState('')
    const [error, setError] = useState('')
    const [importStep, setImportStep] = useState<ImportStep>('idle')
    const [busy, setBusy] = useState(false)

    useEffect(() => {
        api.GetSettings().then(s => {
            setCheckHours(String(s.checkIntervalHours))
            setKeep(String(s.backupsKeep))
        })
    }, [])

    const run = async (fn: () => Promise<string | void>) => {
        setBusy(true)
        setError('')
        setMessage('')
        try {
            const msg = await fn()
            if (msg) setMessage(msg)
        } catch (e) {
            setError(errorMessage(e))
        } finally {
            setBusy(false)
        }
    }

    const save = () => run(async () => {
        await api.SaveSettings({checkIntervalHours: Number(checkHours), backupsKeep: Number(keep)} as Settings)
        return 'Settings saved.'
    })

    const exportList = () => run(async () => {
        const path = await api.ExportLibrary()
        return path ? `Saved your list to ${path}` : ''
    })

    const importList = (mode: 'merge' | 'replace') => run(async () => {
        setImportStep('idle')
        const r = await api.ImportLibrary(mode)
        if (r.added + r.updated + r.skipped === 0) return ''
        onImported()
        return `Imported: ${r.added} added, ${r.updated} updated, ${r.skipped} unchanged. Covers and latest chapters will fill in shortly.`
    })

    return (
        <Modal title="Settings" onClose={onClose} wide>
            <form className="form" onSubmit={e => {
                e.preventDefault()
                save()
            }}>
                <div className="settings-grid">
                    <label>Check for new chapters every
                        <span className="with-unit">
                            <input type="number" min={1} max={168} value={checkHours} onChange={e => setCheckHours(e.target.value)}/>
                            hours
                        </span>
                    </label>
                    <label>Daily backups to keep
                        <span className="with-unit">
                            <input type="number" min={1} max={50} value={keep} onChange={e => setKeep(e.target.value)}/>
                            days
                        </span>
                    </label>
                </div>
                <div className="actions"><button type="submit" className="primary" disabled={busy}>Save settings</button></div>
            </form>

            <section className="panel-section">
                <h3>Your list</h3>
                <p className="faint small">Export saves everything to a single file you can keep anywhere. Import reads one back.</p>
                <div className="button-row">
                    <button onClick={exportList} disabled={busy}>Export list…</button>
                    {importStep === 'idle' && <button onClick={() => setImportStep('choose')} disabled={busy}>Import list…</button>}
                </div>
                {importStep === 'choose' && (
                    <div className="confirm">
                        <span>How should the file be imported?</span>
                        <button onClick={() => importList('merge')}>Merge into my list</button>
                        <button className="danger" onClick={() => setImportStep('confirm-replace')}>Replace my list</button>
                        <button className="ghost" onClick={() => setImportStep('idle')}>Cancel</button>
                    </div>
                )}
                {importStep === 'confirm-replace' && (
                    <div className="confirm">
                        <span className="error-text">This replaces your whole list with the file's contents.</span>
                        <button className="danger" onClick={() => importList('replace')}>Yes, replace it</button>
                        <button className="ghost" onClick={() => setImportStep('idle')}>Cancel</button>
                    </div>
                )}
            </section>

            <section className="panel-section">
                <h3>Data folder</h3>
                <p className="small data-dir">{dataDir}</p>
                <p className="faint small">Copy the whole ChapterMark folder (the exe and this data folder) to move your list to another PC.</p>
                <div className="button-row"><button onClick={() => api.OpenDataFolder()}>Open data folder</button></div>
            </section>

            {message && <p className="ok-text">{message}</p>}
            {error && <p className="error-text">{error}</p>}
            <p className="faint small version">ChapterMark {version}</p>
        </Modal>
    )
}
