export namespace checker {
	
	export class Progress {
	    running: boolean;
	    done: number;
	    total: number;
	    failed: number;
	    // Go type: time
	    lastCheckedAt: any;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.done = source["done"];
	        this.total = source["total"];
	        this.failed = source["failed"];
	        this.lastCheckedAt = this.convertValues(source["lastCheckedAt"], null);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace exportimport {
	
	export class Result {
	    added: number;
	    updated: number;
	    skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	    }
	}

}

export namespace library {
	
	export class AddInput {
	    status: string;
	    myChapter: number;
	    myRating: number;
	    typeOverride: string;
	
	    static createFrom(source: any = {}) {
	        return new AddInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.myChapter = source["myChapter"];
	        this.myRating = source["myRating"];
	        this.typeOverride = source["typeOverride"];
	    }
	}
	export class EntryUpdate {
	    status: string;
	    myChapter: number;
	    myRating: number;
	    notes: string;
	    readUrl: string;
	    typeOverride: string;
	    latestManual?: number;
	
	    static createFrom(source: any = {}) {
	        return new EntryUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.myChapter = source["myChapter"];
	        this.myRating = source["myRating"];
	        this.notes = source["notes"];
	        this.readUrl = source["readUrl"];
	        this.typeOverride = source["typeOverride"];
	        this.latestManual = source["latestManual"];
	    }
	}
	export class ItemView {
	    series: store.Series;
	    entry: store.Entry;
	    effectiveType: string;
	    effectiveLatest: number;
	    newCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.series = this.convertValues(source["series"], store.Series);
	        this.entry = this.convertValues(source["entry"], store.Entry);
	        this.effectiveType = source["effectiveType"];
	        this.effectiveLatest = source["effectiveLatest"];
	        this.newCount = source["newCount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OnlineResult {
	    id: number;
	    title: string;
	    hitTitle: string;
	    type: string;
	    mappedType: string;
	    year: string;
	    url: string;
	    coverUrl: string;
	    rating: number;
	    inLibrary: boolean;
	    seriesId: number;
	
	    static createFrom(source: any = {}) {
	        return new OnlineResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.hitTitle = source["hitTitle"];
	        this.type = source["type"];
	        this.mappedType = source["mappedType"];
	        this.year = source["year"];
	        this.url = source["url"];
	        this.coverUrl = source["coverUrl"];
	        this.rating = source["rating"];
	        this.inLibrary = source["inLibrary"];
	        this.seriesId = source["seriesId"];
	    }
	}

}

export namespace main {
	
	export class InitResult {
	    ready: boolean;
	    error: string;
	    dataDir: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new InitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.error = source["error"];
	        this.dataDir = source["dataDir"];
	        this.version = source["version"];
	    }
	}
	export class Settings {
	    checkIntervalHours: number;
	    backupsKeep: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checkIntervalHours = source["checkIntervalHours"];
	        this.backupsKeep = source["backupsKeep"];
	    }
	}

}

export namespace store {
	
	export class Entry {
	    seriesId: number;
	    status: string;
	    myChapter: number;
	    myRating: number;
	    notes: string;
	    readUrl: string;
	    // Go type: time
	    lastReadAt: any;
	    // Go type: time
	    addedAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seriesId = source["seriesId"];
	        this.status = source["status"];
	        this.myChapter = source["myChapter"];
	        this.myRating = source["myRating"];
	        this.notes = source["notes"];
	        this.readUrl = source["readUrl"];
	        this.lastReadAt = this.convertValues(source["lastReadAt"], null);
	        this.addedAt = this.convertValues(source["addedAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Series {
	    id: number;
	    muId: number;
	    mdId: string;
	    alId: number;
	    malId: number;
	    title: string;
	    romanizedTitle: string;
	    nativeTitle: string;
	    muUrl: string;
	    year: string;
	    pubStatus: string;
	    type: string;
	    typeOverride: string;
	    completed: boolean;
	    coverFile: string;
	    coverUrl: string;
	    muRating: number;
	    muVotes: number;
	    alRating: number;
	    latestChapter: number;
	    latestRaw: string;
	    latestGroup: string;
	    latestDate: string;
	    latestManual?: number;
	    // Go type: time
	    detailsRefreshedAt: any;
	    // Go type: time
	    chaptersCheckedAt: any;
	    altTitles: string[];
	
	    static createFrom(source: any = {}) {
	        return new Series(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.muId = source["muId"];
	        this.mdId = source["mdId"];
	        this.alId = source["alId"];
	        this.malId = source["malId"];
	        this.title = source["title"];
	        this.romanizedTitle = source["romanizedTitle"];
	        this.nativeTitle = source["nativeTitle"];
	        this.muUrl = source["muUrl"];
	        this.year = source["year"];
	        this.pubStatus = source["pubStatus"];
	        this.type = source["type"];
	        this.typeOverride = source["typeOverride"];
	        this.completed = source["completed"];
	        this.coverFile = source["coverFile"];
	        this.coverUrl = source["coverUrl"];
	        this.muRating = source["muRating"];
	        this.muVotes = source["muVotes"];
	        this.alRating = source["alRating"];
	        this.latestChapter = source["latestChapter"];
	        this.latestRaw = source["latestRaw"];
	        this.latestGroup = source["latestGroup"];
	        this.latestDate = source["latestDate"];
	        this.latestManual = source["latestManual"];
	        this.detailsRefreshedAt = this.convertValues(source["detailsRefreshedAt"], null);
	        this.chaptersCheckedAt = this.convertValues(source["chaptersCheckedAt"], null);
	        this.altTitles = source["altTitles"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

