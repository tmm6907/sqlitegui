<script lang="ts">
    import ResultAlert from "./ResultAlert.svelte";
    import { UpdateDB } from "../../wailsjs/go/main/App.js";
    import { appState } from "src/stores/appState.svelte.ts";
    import { triggerResultAlert } from "src/utils/utils.ts";

    let prevEdit = $state({ id: "", value: "" });
    async function handleEdit(
        value: string,
        rowIndex: number,
        colIndex: number,
    ) {
        let rowID = appState.queryResults.rows[rowIndex][0];
        let col = appState.queryResults.cols[colIndex];
        let rowIDName = appState.queryResults.cols[0];

        if (rowID !== prevEdit.id || value !== prevEdit.value) {
            let query = `UPDATE ${appState.currentDB}.${appState.selectedTable} SET ${col} = '%s' WHERE ${rowIDName} = '%v';`;
            prevEdit.id = rowID;
            prevEdit.value = value;

            let res = await UpdateDB({ query: query, id: rowID, value: value });
            if (res.error !== "") {
                console.error("update error", res.error);
            }
            triggerResultAlert(
                `${appState.currentDB}.${appState.selectedTable} updated successfully!`,
            );
        }
    }
</script>

<div class="flex flex-col gap-1 h-full">
    <div class="flex relative justify-between items-center">
        <span
            >Results{#if appState.queryResults.editable}
                <span class="text-neutral-400 ml-2">(Editable)</span>
            {/if}</span
        >
        <ResultAlert />
    </div>
    <div
        class={`
        rounded-box bg-base-100
        outline ${appState.queryResults.editable ? "outline-warning" : "outline-base-content"} 
        flex-1 overflow-y-auto overflow-x-auto 
        h-full p-[1em]
        `}
    >
        <table
            id="main-db"
            class="table table-sm table-pin-rows active text-lg w-full"
        >
            <thead class="text-center">
                <tr>
                    {#each appState.queryResults.cols as col, i}
                        <th class=""
                            >{i == 0 && appState.queryResults.pk
                                ? `${col} ( pk )`
                                : col}</th
                        >
                    {/each}
                </tr>
            </thead>
            <tbody class="text-center">
                {#each appState.queryResults.rows as row, rowIndex}
                    <tr class="hover">
                        {#each row as item, colIndex}
                            {#if colIndex == 0 || !appState.queryResults.editable}
                                <td id={item}>{item}</td>
                            {:else}
                                <td>
                                    <input
                                        class="input focus:input-accent text-center"
                                        type="text"
                                        onblur={async (e) =>
                                            handleEdit(
                                                (e.target as HTMLInputElement)
                                                    .value,
                                                rowIndex,
                                                colIndex,
                                            )}
                                        title={item}
                                        value={item}
                                    />
                                </td>
                            {/if}
                        {/each}
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
</div>
