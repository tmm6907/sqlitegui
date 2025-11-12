export namespace main {
	
	export class AppResult {
	    error: any;
	    results: any;
	
	    static createFrom(source: any = {}) {
	        return new AppResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.error = source["error"];
	        this.results = source["results"];
	    }
	}
	export class CreateDBRequest {
	    name: string;
	    cache: string;
	    journal: string;
	    sync: string;
	    lock: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateDBRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.cache = source["cache"];
	        this.journal = source["journal"];
	        this.sync = source["sync"];
	        this.lock = source["lock"];
	    }
	}
	export class QueryRequest {
	    query: string;
	    editable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.editable = source["editable"];
	    }
	}
	export class UpdateRequest {
	    db: string;
	    table: string;
	    row: any[][];
	    column: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.db = source["db"];
	        this.table = source["table"];
	        this.row = source["row"];
	        this.column = source["column"];
	        this.value = source["value"];
	    }
	}

}

