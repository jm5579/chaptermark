import {useCallback, useEffect, useMemo, useState} from 'react'
import {api, errorMessage, onEvent, type InitResult, type ItemView, type OnlineResult, type Progress} from './lib/api'
import {useNow} from './lib/hooks'
import {selectView, STATUS_LABELS, TYPE_LABELS, visibleItems, type SortMode, type StatusKey, type TypeKey} from './lib/view'
import {TopBar} from './components/TopBar'
import {EmptyState, SeriesRow, Shelf, StatusTabs, TypeSwitch} from './components/Home'
import {SearchPanel} from './components/SearchPanel'
import {AddDialog} from './components/AddDialog'
import {DetailPanel} from './components/DetailPanel'
import {SettingsDialog} from './components/SettingsDialog'
import {ListToolbar, type ListFilters} from './components/ListToolbar'

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
    const [adding, setAdding] = useState<OnlineResult | null>(null)
    const [settingsOpen, setSettingsOpen] = useState(false)
    const [filters, setFilters] = useState<ListFilters>({sort: 'smart', staleOnly: false, tag: ''})
    const [showAdult, setShowAdult] = useState(false)
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
            // Before the list loads, so 18+ series never flash up.
            setShowAdult((await api.GetSettings()).showAdult)
            await reload()
            setProgress(await api.CheckStatus())
            const p = await api.GetViewPrefs()
            setFilters({sort: p.sort as SortMode, staleOnly: p.staleOnly, tag: p.tag})
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

    const onAdded = (v: ItemView) => {
        setItems(prev => upsert(prev, v))
        setAdding(null)
        if (v.adult && !showAdult) {
            // MangaDex or AniList can mark a series 18+ that MangaUpdates' genres didn't.
            setToast(`Added ${v.series.title}. It's marked 18+, so it stays hidden until you allow 18+ titles in Settings.`)
            return
        }
        setType(v.effectiveType as TypeKey)
        setTab(v.entry.status as StatusKey)
        setToast(`Added ${v.series.title}`)
    }

    const changeFilters = (f: ListFilters) => {
        setFilters(f)
        api.SaveViewPrefs(f as any).catch(e => setToast(errorMessage(e)))
    }

    const visible = useMemo(() => visibleItems(items, showAdult), [items, showAdult])
    const view = useMemo(() => selectView(items, type, tab, {...filters, showAdult, now}), [items, type, tab, filters, showAdult, now])
    const allTags = useMemo(() => [...new Set(visible.flatMap(it => it.entry.tags ?? []))]
        .sort((a, b) => a.localeCompare(b, undefined, {sensitivity: 'base'})), [visible])
    const filtering = filters.staleOnly || filters.tag !== ''
    const openItem = openId === null ? undefined : items.find(x => x.series.id === openId)

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
                search={<SearchPanel showAdult={showAdult} onOpen={setOpenId} onAdd={setAdding}/>}
                progress={progress}
                now={now}
                onCheck={() => api.CheckNow()}
                onSettings={() => setSettingsOpen(true)}
            />
            <main className="content">
                <TypeSwitch value={type} newCounts={view.newCounts} onChange={setType}/>
                <Shelf items={view.shelf} onOpen={setOpenId}/>
                <StatusTabs value={tab} counts={view.tabCounts} onChange={setTab}/>
                <ListToolbar value={filters} tagCounts={view.tagCounts} onChange={changeFilters}/>
                <div className="series-list">
                    {view.list.length === 0
                        ? <EmptyState text={visible.length === 0
                            ? 'Nothing here yet. Search above to add your first series.'
                            : filtering
                                ? 'Nothing matches these filters.'
                                : `No ${TYPE_LABELS[type].toLowerCase()} in "${STATUS_LABELS[tab]}".`}/>
                        : view.list.map(it => <SeriesRow key={it.series.id} it={it} now={now} onOpen={setOpenId}/>)}
                </div>
            </main>
            {settingsOpen && <SettingsDialog dataDir={init.dataDir} version={init.version}
                                             onClose={() => setSettingsOpen(false)}
                                             onSaved={s => setShowAdult(s.showAdult)}
                                             onImported={reload} onTagsChanged={reload}/>}
            {adding && <AddDialog result={adding} onClose={() => setAdding(null)} onAdded={onAdded}/>}
            {toast && <div className="toast" role="status">{toast}</div>}
            {openItem && (
                <DetailPanel item={openItem} now={now} allTags={allTags} onClose={() => setOpenId(null)}
                             onChange={v => setItems(prev => upsert(prev, v))}
                             onRemoved={id => {
                                 setItems(prev => prev.filter(x => x.series.id !== id))
                                 setOpenId(null)
                                 setToast(`Removed ${openItem.series.title}`)
                             }}/>
            )}
        </div>
    )
}

export default App
