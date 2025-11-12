import { appState, type AlertType, type DialogMessage, type QueryResults } from "src/stores/appState.svelte.ts"
import { GetCurrentDB, GetNavData } from "../../wailsjs/go/main/App.js"

export const setQueryResults = (
    results: QueryResults = { pk: false, cols: [], rows: [], editable: false }
) => {
    appState.queryResults.pk = results.pk
    appState.queryResults.cols = results.cols
    appState.queryResults.rows = results.rows
    appState.queryResults.editable = results.editable
}

export const setDialogMsg = (msg: DialogMessage) => {
    appState.dialog.title = msg.title
    appState.dialog.msg = msg.msg
    appState.dialog.options = msg.options
    appState.dialog.actions = msg.actions
    appState.dialog.show = msg.show
    appState.dialog.btnStyles = msg.btnStyles
}

export function triggerAlert(msg: string, type: AlertType = "success", duration = 3000) {
    appState.alert.msg = msg
    appState.alert.type = type
    appState.alert.duration = duration
    appState.alert.show = true
}
export function triggerResultAlert(msg: string, type: AlertType = "success", duration = 5000) {
    appState.resultAlert.msg = msg
    appState.resultAlert.type = type
    appState.resultAlert.duration = duration
    appState.resultAlert.show = true
}

export async function renderNav() {
    let res = await GetNavData();
    if (res.error) {
        triggerAlert(res.error, "error")
        return
    }
    console.log("Nav:", res)
    appState.navData = res.results ? res.results : []
    res = await GetCurrentDB();
    if (res.error) {
        triggerAlert(res.error, "error");
    }
    appState.currentDB = res.results ? res.results : "";
}

export async function renderNavWithAlert(msg: string, type: AlertType = "success") {
    await renderNav();
    triggerAlert(msg, type);
}

export async function renderNavWithResultAlert(msg: string, type: AlertType = "success") {
    await renderNav();
    triggerResultAlert(msg, type);
}