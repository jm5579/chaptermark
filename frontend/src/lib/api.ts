// Thin typed layer over the Wails-generated bindings.
import * as App from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import type {checker, exportimport, library, main, store} from '../../wailsjs/go/models'

export type ItemView = library.ItemView
export type OnlineResult = library.OnlineResult
export type AddInput = library.AddInput
export type EntryUpdate = library.EntryUpdate
export type Progress = checker.Progress
export type Settings = main.Settings
export type InitResult = main.InitResult
export type ImportResult = exportimport.Result
export type ViewPrefs = main.ViewPrefs
export type Tag = store.Tag

export const api = App

/** Subscribe to a backend event; returns an unsubscribe function. */
export function onEvent<T>(name: 'check:progress' | 'library:item', cb: (data: T) => void): () => void {
    return EventsOn(name, cb)
}

/** URL of a cached cover image served by the Go backend. */
export function coverSrc(file: string | undefined): string {
    return file ? `/covers/${file}` : ''
}

/** Wails rejects promises with the Go error string. */
export function errorMessage(e: unknown): string {
    if (typeof e === 'string') return e
    if (e instanceof Error) return e.message
    return String(e)
}

/** The full editable state of an entry, as UpdateEntry expects it. */
export function toUpdate(it: ItemView, patch: Partial<EntryUpdate> = {}): EntryUpdate {
    return {
        status: it.entry.status,
        myChapter: it.entry.myChapter,
        myRating: it.entry.myRating,
        notes: it.entry.notes,
        readUrl: it.entry.readUrl,
        typeOverride: it.series.typeOverride,
        latestManual: it.series.latestManual,
        tags: it.entry.tags ?? [],
        ...patch,
    } as EntryUpdate
}
