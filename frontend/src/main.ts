// Svelte 5 Instantiation
import { mount } from 'svelte';
import App from "./App.svelte";
import './style.css';
import { EventsOn } from "../wailsjs/runtime/runtime.js";
import { renderNavDataWithAlert } from "./stores/renderNav.ts";

const app = mount(App, {
  target: document.getElementById('app'),
  props: {
    // ...
  }
});

interface emitData {
  msg: string
}

export function setupWailsEventsListeners() {
  EventsOn("dbAttached", async () => {
    await renderNavDataWithAlert("DB imported successfully!");
  });
  EventsOn("dbAttachFailed", async () => {
    await renderNavDataWithAlert("DB failed to import!", "alert-error");
  });
  EventsOn("dbExportFailed", async (data) => {
    await renderNavDataWithAlert(data.error, "alert-error")
  })
  EventsOn("dbExportSucceeded", async (data: emitData) => {
    await renderNavDataWithAlert(data.msg)
  })

  EventsOn("dbUploadFailed", async (data) => {
    await renderNavDataWithAlert(data.error, "alert-error")
  })

  EventsOn("dbUploadSucceeded", async (data: emitData) => {
    await renderNavDataWithAlert(data.msg)
  })
}