import { writable } from "svelte/store";
import { triggerAlert } from "./alertStore.ts";
import { GetNavData } from "../../wailsjs/go/main/App.js";

export const navDataStore = writable({
    databases: {}
});
export const rootDBPathStore = writable("")

export async function renderNavData() {
    let res = await GetNavData();
    navDataStore.set({ databases: res.results ? res.results : [] });
}

export async function renderNavDataWithAlert(msg: string, type: string = "alert-success") {
    await renderNavData();
    triggerAlert(msg, type);
}
