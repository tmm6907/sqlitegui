<script lang="ts">
    import { onDestroy, onMount } from "svelte";
    import {
        navDataStore,
        renderNavDataWithAlert,
        renderNavData,
    } from "../stores/renderNav.ts";
    import {
        CreateDB,
        SetCurrentDB,
        GetCurrentDB,
        Query,
    } from "../../wailsjs/go/main/App.js";
    import { triggerAlert } from "src/stores/alertStore.ts";
    import {
        queryResults,
        tableName,
        dbNameStore,
        loadingResultsStore,
    } from "../stores/resultsStore.ts";

    var modal: HTMLDialogElement;

    let openDBName = $state("");

    let databases = $state({});

    navDataStore.subscribe((data) => {
        databases = data.databases;
    });

    async function selectAll(db: string, table: string) {
        let query = `SELECT * FROM ${db}.${table} LIMIT 50;`;
        loadingResultsStore.set(true);
        let res = await Query({ query: query, editable: true });
        loadingResultsStore.set(false);
        if (res.error) {
            console.error(res.error);
        }
        let results = res.results;
        queryResults.set({
            pk: results.pk,
            cols: results.cols,
            rows: results.rows,
            editable: true,
        });
        tableName.set(table);
    }

    function openDB() {
        if (modal) {
            modal.showModal();
        }
    }

    async function createDB(form: HTMLFormElement) {
        var getRadioValue = (name: string) => {
            const selectedOption: HTMLInputElement = form.querySelector(
                `input[name='${name}']:checked`,
            );
            return selectedOption ? selectedOption.value : "";
        };
        const nameInput: HTMLInputElement =
            form.querySelector("input[name='name']");
        const cache = getRadioValue("cache");
        const journal = getRadioValue("journal");
        const sync = getRadioValue("sync");
        const lock = getRadioValue("lock");
        var formData = {
            name: nameInput.value,
            cache: cache,
            journal: journal,
            sync: sync,
            lock: lock,
        };
        CreateDB(formData).then((res) => {
            if (res.error !== "") {
                console.error(res.error);
                triggerAlert("DB failed to be created!", "alert-error");
            } else {
                modal.close();
                renderNavDataWithAlert("DB created successfully!");
            }
        });
    }
    function refreshSchema() {
        navDataStore.set({ databases: {} });
        renderNavDataWithAlert("Schema refreshed successfully!");
    }

    async function handleToggle(e: MouseEvent, dbName: string) {
        e.preventDefault();
        if (openDBName === dbName) {
            openDBName = ""; // Sets 'open' property to false on the current element
            dbNameStore.set("");
            return;
        }

        openDBName = dbName;

        let res = await SetCurrentDB(openDBName);

        if (res.error !== "") {
            console.error(res.error);
            triggerAlert(res.error, "alert-error");
            openDBName = "";
        } else {
            dbNameStore.set(openDBName);
        }
    }

    const handleSubmit = (e: SubmitEvent) => {
        if (e.target instanceof HTMLFormElement && e.target.id === "db-form") {
            e.preventDefault();
            createDB(e.target);
            renderNavData();
        }
    };

    // Show the modal on mount
    onMount(async () => {
        // Update sessionStorage whenever openDBName changes
        let res = await GetCurrentDB();
        if (res.error !== "") {
            triggerAlert(res.error, "alert-error");
        }
        openDBName = res.results ? res.results : "";
        dbNameStore.set(openDBName);
        renderNavData();
        document.addEventListener("submit", handleSubmit);
    });
    onDestroy(() => document.removeEventListener("submit", handleSubmit));
</script>

<dialog bind:this={modal} class="modal">
    <div class="modal-box">
        <form id="db-form" action="" method="POST">
            <h3 class="text-secondary modal-header text-2xl pt-4 pb-8">
                Create Database
            </h3>
            <div class="grid grid-cols-1 gap-y-4 max-w-[64ch]">
                <label class="label cursor-pointer text-accent"
                    ><span>DB Name:</span><input
                        type="text"
                        name="name"
                        class="outline outline-neutral-content rounded"
                    /></label
                >
                <label class="label cursor-pointer text-accent"
                    ><span>Connection Cache:</span>
                    <div class="flex gap-6">
                        <label
                            for="cache-private"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Private</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="cache"
                                id="cache-private"
                                checked
                            /></label
                        >
                        <label
                            for="cache-shared"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Shared</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="cache"
                                id="cache-shared"
                            /></label
                        >
                    </div>
                </label>
                <label class="label cursor-pointer text-accent"
                    ><span>Journal Mode:</span>
                    <div class="flex gap-6">
                        <label
                            for="journal-normal"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Normal</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="journal"
                                id="journal-normal"
                                checked
                            /></label
                        >
                        <label
                            for="journal-wal"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">WAL</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="journal"
                                id="journal-wal"
                            /></label
                        >
                    </div>
                </label>
                <label class="label cursor-pointer text-accent"
                    ><span>Synchronous Mode:</span>
                    <div class="flex gap-6">
                        <label
                            for="sync-normal"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Normal</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="sync"
                                id="sync-normal"
                                checked
                            /></label
                        >
                        <label
                            for="sync-full"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Full</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="sync"
                                id="sync-full"
                            /></label
                        >
                        <label
                            for="sync-off"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Off</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="sync"
                                id="sync-off"
                            /></label
                        >
                    </div>
                </label>
                <label class="label cursor-pointer text-accent"
                    ><span>Locking Mode:</span>
                    <div class="flex gap-6">
                        <label
                            for="lock-normal"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Normal</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="lock"
                                id="lock-normal"
                                checked
                            /></label
                        >
                        <label
                            for="lock-exclusive"
                            class="label cursor-pointer text-neutral-content"
                            ><span class="mr-2">Exclusive</span><input
                                type="radio"
                                class="radio radio-primary"
                                name="lock"
                                id="lock-exclusive"
                            /></label
                        >
                    </div>
                </label>
            </div>
            <div class="modal-action btn-group">
                <button
                    id="cancel-db-btn"
                    class="btn"
                    type="button"
                    onclick={() => modal.close()}>Cancel</button
                >
                <button id="create-db-btn" class="btn btn-primary" type="submit"
                    >Create</button
                >
            </div>
        </form>
    </div>
</dialog>

<nav class="h-screen w-full bg-base-200 py-8 px-3 flex flex-col gap-2">
    <div class="flex items-center justify-between space-x-8">
        <div class="flex items-center space-x-2">
            <span>Schemas</span>
            <button
                class="btn btn-xs btn-ghost"
                aria-label="refresh schemas"
                onclick={refreshSchema}
                ><i class="fa-solid fa-arrows-rotate self-center"></i></button
            >
        </div>
        <button class="btn btn-sm btn-ghost" onclick={openDB}
            ><i class="fa-solid fa-plus"></i><span>New DB</span>
        </button>
    </div>

    <ul class="menu menu-vertical w-full">
        <li id="nav-item-list">
            {#if Object.keys(databases).length > 0}
                <details open={"main" === openDBName} class="nav-menu-item">
                    <summary
                        class="truncate text-secondary"
                        title={"main"}
                        onclick={(e) => handleToggle(e, "main")}
                        ><i class="fa-solid fa-database"></i>{"main"}</summary
                    >
                    <ul>
                        {#if databases["main"]}
                            {#each databases["main"] as tblName}
                                <li>
                                    <div class="flex space-x-1">
                                        <i class="fa-solid fa-table"></i>
                                        <span class="truncate" title={tblName}
                                            >{tblName}
                                        </span>
                                        <button
                                            class="btn btn-xs btn-ghost"
                                            aria-label={`Edit ${tblName}`}
                                            onclick={async () =>
                                                await selectAll(
                                                    databases["main"],
                                                    tblName,
                                                )}
                                            ><i class="fa-solid fa-pencil"
                                            ></i></button
                                        >
                                    </div>
                                </li>
                            {/each}
                        {:else}
                            <li>No tables found</li>
                        {/if}
                    </ul>
                </details>
                {#each Object.keys(databases) as dbName}
                    {#if dbName !== "main"}
                        <details
                            open={dbName === openDBName}
                            class="nav-menu-item"
                        >
                            <summary
                                class="truncate"
                                title={dbName}
                                onclick={(e) => handleToggle(e, dbName)}
                                ><i class="fa-solid fa-database"
                                ></i>{dbName}</summary
                            >
                            <ul>
                                {#if databases[dbName]}
                                    {#each databases[dbName] as tblName}
                                        <li>
                                            <div class="flex space-x-1">
                                                <i class="fa-solid fa-table"
                                                ></i>
                                                <span
                                                    class="truncate"
                                                    title={tblName}
                                                    >{tblName}
                                                </span>
                                                <button
                                                    class="btn btn-xs btn-ghost"
                                                    aria-label={`Edit ${tblName}`}
                                                    onclick={async () =>
                                                        await selectAll(
                                                            dbName,
                                                            tblName,
                                                        )}
                                                    ><i
                                                        class="fa-solid fa-pencil"
                                                    ></i></button
                                                >
                                            </div>
                                        </li>
                                    {/each}
                                {:else}
                                    <li>No tables found</li>
                                {/if}
                            </ul>
                        </details>
                    {/if}
                {/each}
            {:else}
                <div class="flex justify-center">
                    <span class="loading loading-spinner loading-md mx-auto"
                    ></span>
                    <span>Loading Schema</span>
                </div>
            {/if}
        </li>
    </ul>
</nav>

<style>
</style>
