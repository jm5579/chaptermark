import {useCallback, useEffect, useMemo, useState} from 'react'
import {api, errorMessage, onEvent, type InitResult, type ItemView, type Progress} from './lib/api'
import {useNow} from './lib/hooks'
import {selectView, STATUS_LABELS, TYPE_LABELS, type StatusKey, type TypeKey} from './lib/view'
import {TopBar} from './components/TopBar'
import {EmptyState, SeriesRow, Shelf, StatusTabs, TypeSwitch} from './components/Home'

function upsert(items: ItemView[], v: ItemView): ItemView[] {
    const i = items.findIndex(x => x.series.id === v.series.id)
    if (i < 0) return [...items, v]
    const next = items.slice()
    next[i] = v
    return next
}

function App() {
    const [init, setInit] = useState<InitResult | null>(null)
    const [items, setItems] = useState<ItemView[]>([])
    const [type, setType] = useState<TypeKey>('manhwa')
    const [tab, setTab] = useState<StatusKey>('reading')
    const [progress, setProgress] = useState<Progress | null>(null)
    const [openId, setOpenId] = useState<number | null>(null)
    const [toast, setToast] = useState<string | null>(null)
    const now = useNow()

    const reload = useCallback(async () => {
        try {
            setItems(await api.ListLibrary() ?? [])
        } catch (e) {
            setToast(errorMessage(e))
        }
    }, [])

    useEffect(() => {
        api.Init().then(async res => {
            setInit(res)
            if (!res.ready) return
            await reload()
            setProgress(await api.CheckStatus())
        })
        const offItem = onEvent<ItemView>('library:item', v => setItems(prev => upsert(prev, v)))
        const offProgress = onEvent<Progress>('check:progress', setProgress)
        return () => {
            offItem()
            offProgress()
        }
    }, [reload])

    useEffect(() => {
        if (!toast) return
        const id = setTimeout(() => setToast(null), 3500)
        return () => clearTimeout(id)
    }, [toast])

    const view = useMemo(() => selectView(items, type, tab), [items, type, tab])

    if (!init) return <div className="splash">ChapterMark</div>
    if (!init.ready) {
        return (
            <div className="fatal">
                <h1>ChapterMark can't start</h1>
                <p>{init.error}</p>
                <p className="faint">Data folder: {init.dataDir}</p>
            </div>
        )
    }

    return (
        <div className="app">
            <TopBar
                search={<input className="search-input" placeholder="Search your list or add a new series…" disabled/>}
                progress={progress}
                now={now}
                onCheck={() => api.CheckNow()}
                onSettings={() => undefined}
            />
            <main className="content">
                <TypeSwitch value={type} newCounts={view.newCounts} onChange={setType}/>
                <Shelf items={view.shelf} onOpen={setOpenId}/>
                <StatusTabs value={tab} counts={view.tabCounts} onChange={setTab}/>
                <div className="series-list">
                    {view.list.length === 0
                        ? <EmptyState text={items.length === 0
                            ? 'Nothing here yet — search above to add your first series.'
                            : `No ${TYPE_LABELS[type].toLowerCase()} in “${STATUS_LABELS[tab]}”.`}/>
                        : view.list.map(it => <SeriesRow key={it.series.id} it={it} now={now} onOpen={setOpenId}/>)}
                </div>
            </main>
            {toast && <div className="toast" role="status">{toast}</div>}
            {openId !== null && null}
        </div>
    )
}

export default App
