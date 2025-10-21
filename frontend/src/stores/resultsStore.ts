import { writable } from 'svelte/store';

export const queryResults = writable(
    {
        pk: false,
        cols: [],
        rows: [],
        editable: false,
    }
);

export const tableName = writable("")
export const dbNameStore = writable("")
export const loadingResultsStore = writable(false)