<script lang="ts">
    import { onDestroy, onMount } from "svelte";
    import {
        CreateDB,
        SetCurrentDB,
        GetCurrentDB,
        RemoveDB,
        QueryAll,
    } from "../../wailsjs/go/main/App.js";
    import { appState } from "src/stores/appState.svelte.ts";
    import {
        renderNav,
        renderNavWithAlert,
        setDialogMsg,
        setQueryResults,
        triggerAlert,
    } from "src/utils/utils.ts";

    var modal: HTMLDialogElement;
    async function selectAll(table: string) {
        appState.loadingQueryResults = true;
        let res = await QueryAll(table);
        appState.loadingQueryResults = false;
        console.log(res);
        if (res.error) {
            console.error(res.error);
        }
        setQueryResults(res.results);
        appState.selectedTable = table;
    }

    function openDB() {
        if (modal) {
            modal.showModal();
        }
    }

    async function createDB(e: SubmitEvent) {
        e.preventDefault();
        const form = e.target as HTMLFormElement;
        const formData = new FormData(form);
        let data = Object.fromEntries(formData.entries());
        console.log(data);
        let res = await CreateDB(data);
        if (res.error) {
            console.error(res.error);
            triggerAlert("DB failed to be created!", "error");
            return;
        }
        modal.close();
        await renderNavWithAlert("DB created successfully!");
    }
    async function refreshSchema() {
        appState.navData = {};
        await renderNavWithAlert("Schema refreshed successfully!");
    }

    async function handleToggle(e: MouseEvent, dbName: string) {
        e.preventDefault();
        appState.queryResults.editable = false;
        if (appState.currentDB === dbName) {
            appState.currentDB = "";
            return;
        }
        appState.currentDB = dbName;

        let res = await SetCurrentDB(appState.currentDB);

        if (res.error) {
            console.error(res.error);
            triggerAlert(res.error, "error");
            appState.currentDB = "";
        }
    }

    const handleSubmit = async (e: SubmitEvent) => {
        if (e.target instanceof HTMLFormElement && e.target.id === "db-form") {
            await createDB(e);
        }
    };

    async function removeDB(name: string) {
        setDialogMsg({
            title: "Are you sure?",
            msg: `Are you sure you want to remove '${name}' db? THIS ACTION CANNOT BE UNDONE!`,
            options: ["Cancel", `Remove '${name}'`],
            actions: [
                () => {},
                async () => {
                    let res = await RemoveDB(name);
                    if (res.error) {
                        triggerAlert(res.error, "error");
                        return;
                    }
                    await renderNavWithAlert(`Removed '${name}' successfully!`);
                },
            ],
            show: true,
            btnStyles: ["btn-neutral", "btn-primary"],
        });
    }

    // Show the modal on mount
    onMount(async () => {
        // Update sessionStorage whenever openDBName changes
        let res = await GetCurrentDB();
        if (res.error) {
            triggerAlert(res.error, "error");
        }
        appState.currentDB = res.results ? res.results : "";
        await renderNav();
        document.addEventListener("submit", async (e) => await handleSubmit(e));
    });
    onDestroy(() =>
        document.removeEventListener(
            "submit",
            async (e) => await handleSubmit(e),
        ),
    );
</script>

<dialog bind:this={modal} class="modal">
    <div class="modal-box">
        <form id="db-form">
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
                                value="private"
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
                                value="shared"
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
                                value=""
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
                                value="wal"
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
                                value=""
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
                                value="full"
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
                                value="off"
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
                                value=""
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
                                value="exclusive"
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

<nav class="h-screen w-full bg-base-200 py-8 px-3 flex flex-col space-y-2">
    <div class="flex items-center justify-between space-x-2">
        <div class="flex items-center space-x-1">
            <span>Schemas</span>
            <button
                class="btn btn-xs btn-ghost"
                aria-label="refresh schemas"
                onclick={async () => await refreshSchema()}
                ><i class="fa-solid fa-arrows-rotate self-center"></i></button
            >
        </div>
        <button class="btn btn-sm btn-ghost" onclick={openDB}
            ><i class="fa-solid fa-plus"></i><span>New DB</span>
        </button>
    </div>
    <ul class="menu menu-vertical w-full">
        {#if appState.navData && Object.keys(appState.navData).length > 0}
            <li>
                <details open={appState.currentDB === "main"}>
                    <summary
                        class="truncate text-secondary"
                        title={"main"}
                        onclick={(e) => handleToggle(e, "main")}
                        ><i class="fa-solid fa-database"></i>
                        <span>main</span>
                    </summary>

                    <ul>
                        {#if appState.navData["main"]}
                            {#each appState.navData["main"].tables as tblName}
                                <li>
                                    <div class="grid">
                                        <i class="fa-solid fa-table"></i>
                                        <span class="truncate" title={tblName}
                                            >{tblName}
                                        </span>
                                        <button
                                            class="btn btn-xs btn-ghost"
                                            aria-label={`Edit ${tblName}`}
                                            onclick={async () =>
                                                await selectAll(tblName)}
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
            </li>
            <div class="border-b m-2"></div>
            {#if appState.rootPath !== "main"}
                <h3 class="italic truncate w-full" title={appState.rootPath}>
                    {appState.rootPath}
                </h3>
            {/if}
            {#each Object.keys(appState.navData).filter((db) => db !== "main") as db}
                <li>
                    <details
                        open={appState.currentDB === db}
                        class="max-w-full"
                    >
                        <summary
                            title={db}
                            onclick={(e) => handleToggle(e, db)}
                        >
                            <div><i class="fa-solid fa-database"></i></div>
                            <div
                                class="px-2 flex space-x-4 items-center justify-between"
                            >
                                <span class="truncate">{db}</span>
                                {#if appState.navData[db].app_created}
                                    <button
                                        class="btn btn-xs btn-ghost"
                                        aria-label="Remove DB"
                                        onclick={async () => await removeDB(db)}
                                        ><i class="fa-solid fa-trash"></i>
                                    </button>
                                {/if}
                            </div>
                        </summary>
                        <ul>
                            {#if appState.navData[db].tables}
                                {#each appState.navData[db].tables as tblName}
                                    <li>
                                        <div class="grid w-full">
                                            <i class="fa-solid fa-table"></i>
                                            <span
                                                class="truncate"
                                                title={tblName}
                                                >{tblName}
                                            </span>
                                            <button
                                                class="btn btn-xs btn-ghost"
                                                aria-label={`Edit ${tblName}`}
                                                onclick={async () =>
                                                    await selectAll(tblName)}
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
                </li>
            {/each}
        {:else}
            <div class="flex justify-center space-x-2">
                <span class="loading loading-spinner loading-md"></span>
                <span>Loading Schema</span>
            </div>
        {/if}
    </ul>
</nav>

<style>
</style>
