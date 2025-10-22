<script lang="ts">
  import Nav from "./components/Nav.svelte";
  import Editor from "./components/Editor.svelte";
  import Table from "./components/Table.svelte";
  import Alert from "./components/Alert.svelte";
  import { setupWailsEventsListeners } from "./main.ts";
  setupWailsEventsListeners();
</script>

<div class="flex h-screen">
  <Alert />
  <div class="w-2/12">
    <Nav />
  </div>
  <main class=" w-10/12 flex flex-col space-y-4 p-8">
    <div class="h-1/3"><Editor /></div>
    <div class="h-2/3"><Table /></div>
  </main>
</div>

<style>
</style>
