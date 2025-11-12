// Svelte 5 Instantiation
import { mount } from 'svelte';
import App from "./App.svelte";
import './style.css';
import { EventsOn } from "../wailsjs/runtime/runtime.js";
import { renderNavWithAlert } from './utils/utils.ts';

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
    await renderNavWithAlert("DB imported successfully!");
  });
  EventsOn("dbAttachFailed", async () => {
    await renderNavWithAlert("DB failed to import!", "error");
  });
  EventsOn("dbExportFailed", async (data) => {
    await renderNavWithAlert(data.msg, "error")
  })
  EventsOn("dbExportSucceeded", async (data: emitData) => {
    await renderNavWithAlert(data.msg)
  })

  EventsOn("dbUploadFailed", async (data) => {
    await renderNavWithAlert(data.msg, "error")
  })

  EventsOn("dbUploadSucceeded", async (data: emitData) => {
    await renderNavWithAlert(data.msg)
  })

  EventsOn("newWindowFailed", async (data) => {
    await renderNavWithAlert(data.msg, "error")
  })

  EventsOn("newWindowSucceeded", async (data: emitData) => {
    await renderNavWithAlert(data.msg || "New Window Opened!")
  })

  EventsOn("openFolderFailed", async (data) => {
    await renderNavWithAlert(data.msg, "error")
  })

  EventsOn("openFolderSucceeded", async (data: emitData) => {
    await renderNavWithAlert(data.msg || "Database loaded!")
  })
}