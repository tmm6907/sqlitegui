import { writable } from "svelte/store";

export const msgDialogueStore = writable({
    title: "New Dialogue",
    msg: "",
    options: ["Cancel", "OK"],
    actions: [],
    btnStyles: [],
    show: false,
})