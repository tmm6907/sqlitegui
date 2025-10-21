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
  EventsOn("dbAttached", () => {
    renderNavDataWithAlert("DB imported successfully!");
  });
  EventsOn("dbAttachFailed", () => {
    renderNavDataWithAlert("DB failed to import!", "alert-error");
  });

  EventsOn("dbUploadFailed", (data) => {
    renderNavDataWithAlert(data.error, "alert-error")
  })

  EventsOn("dbUploadSucceeded", (data: emitData) => {
    renderNavDataWithAlert(data.msg)
  })
}