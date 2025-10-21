import { writable } from "svelte/store";
import { renderNavData } from "./renderNav.ts";

export const alertStore = writable({
    msg: "",
    type: "alert-success",
    show: false,
    duration: 3000,
});

export const resultAlertStore = writable({
    msg: "",
    type: "success",
    show: false,
    duration: 3000,
});
export function triggerAlert(msg: string, type: string = "alert-success", duration = 3000) {
    alertStore.set({ msg, type, show: true, duration });
    setTimeout(() => {
        alertStore.set({
            msg: "",
            type,
            show: false,
            duration: duration
        });
    }, duration);
}

export function triggerResultAlert(msg: string, type: string = "success", duration = 5000) {
    resultAlertStore.set({ msg, type, show: true, duration });
    setTimeout(() => {
        resultAlertStore.set({
            msg: "",
            type,
            show: false,
            duration: duration
        });
    }, duration);
}

export async function renderNavDataWithResultAlert(msg, type = "success") {
    await renderNavData();
    triggerResultAlert(msg, type);
}
