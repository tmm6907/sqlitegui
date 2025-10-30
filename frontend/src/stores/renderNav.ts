import { writable } from "svelte/store";
import { triggerAlert } from "./alertStore.ts";
import { GetNavData } from "../../wailsjs/go/main/App.js";

export interface DatabaseInfo {
    tables: string[];
    app_created: boolean;
}
export interface NavDatabases {
    [key: string]: DatabaseInfo;
}
export interface NavStore {
    databases: NavDatabases
}
export const navDataStore = writable<NavStore>({
    databases: {}
});
export const rootDBPathStore = writable("")

export async function renderNavData() {
    let res = await GetNavData();
    console.log(res.results)
    navDataStore.set({ databases: res.results ? res.results : [] });
}

export async function renderNavDataWithAlert(msg: string, type: string = "alert-success") {
    await renderNavData();
    triggerAlert(msg, type);
}
