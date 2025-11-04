<script lang="ts">
  import Nav from "./components/Nav.svelte";
  import Editor from "./components/Editor.svelte";
  import Table from "./components/Table.svelte";
  import Alert from "./components/Alert.svelte";
  import { setupWailsEventsListeners } from "./main.ts";
  import { onMount } from "svelte";
  import {
    GetRootPath,
    OpenFolderOnStart,
    SetupMain,
  } from "../wailsjs/go/main/App.js";
  import MessageDialogue from "./components/MessageDialogue.svelte";
  import { appState } from "./stores/appState.svelte.ts";
  setupWailsEventsListeners();
  async function openMain() {
    await SetupMain();
    appState.rootPath = "main";
  }

  async function openFolder() {
    let res = await OpenFolderOnStart();
    if (!res.error) {
      appState.rootPath = res.results.root;
    }
  }

  onMount(async () => {
    let res = await GetRootPath();
    appState.rootPath = res.results.root;
    console.log("Root", appState.rootPath);
  });
</script>

<div class="flex h-screen relative">
  <Alert />
  <MessageDialogue />
  {#if appState.rootPath == ""}
    <div class="flex flex-col space-y-16 flex-1 justify-center items-center">
      <h1 class="text-6xl">SQLite GUI</h1>
      <div class="grid grid-cols-2 gap-4">
        <button
          onclick={async () => await openMain()}
          class="btn btn-link text-xl text-base-content"
        >
          Main Project
        </button>
        <button
          onclick={async () => await openFolder()}
          class="btn btn-link text-xl text-base-content"
        >
          Open Project Folder
        </button>
      </div>
    </div>
  {:else}
    <div class="w-2/12">
      <Nav />
    </div>
    <main class=" w-10/12 flex flex-col space-y-4 p-8">
      <div class="h-1/3"><Editor /></div>
      <div class="h-2/3"><Table /></div>
    </main>
  {/if}
</div>

<style>
</style>
