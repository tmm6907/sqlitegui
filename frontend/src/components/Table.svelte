<script lang="ts">
    import ResultAlert from "./ResultAlert.svelte";
    import { UpdateDB } from "../../wailsjs/go/main/App.js";
    import {
        queryResults,
        tableName,
        dbNameStore,
    } from "../stores/resultsStore.ts";
    import { triggerResultAlert } from "src/stores/alertStore.ts";
    import { title } from "process";
    let rows = $state([]);
    let cols = $state([]);
    let hasPK = $state(false);
    let prevEdit = $state({ id: "", value: "" });
    let editable = $state(false);
    let currTable = $state("");
    let currDB = $state("");
    queryResults.subscribe((value) => {
        rows = value.rows;
        console.log($state.snapshot(rows));
        cols = value.cols;
        hasPK = value.pk;
        editable = value.editable;
    });

    async function handleEdit(
        value: string,
        rowIndex: number,
        colIndex: number,
    ) {
        let rowID = rows[rowIndex][0];
        let col = cols[colIndex];
        let rowIDName = cols[0];
        tableName.subscribe((name) => (currTable = name));
        dbNameStore.subscribe((name) => (currDB = name));
        if (rowID !== prevEdit.id || value !== prevEdit.value) {
            let query = `UPDATE ${currDB}.${currTable} SET ${col} = '%s' WHERE ${rowIDName} = '%v';`;
            prevEdit.id = rowID;
            prevEdit.value = value;

            let res = await UpdateDB({ query: query, id: rowID, value: value });
            if (res.error !== "") {
                console.error("update error", res.error);
            }
            triggerResultAlert(`${currDB}.${currTable} updated successfully!`);
        }
    }
</script>

<div class="flex flex-col gap-1 h-full">
    <div class="flex relative justify-between items-center">
        <span
            >Results{#if editable}
                <span class="text-neutral-500 ml-2">(Editable)</span>
            {/if}</span
        >
        <ResultAlert />
    </div>
    <div
        class={`
        rounded-box bg-base-100
        outline ${editable ? "outline-warning" : "outline-base-content"} 
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
                    {#each cols as col, i}
                        <th class=""
                            >{i == 0 && hasPK ? `${col} ( pk )` : col}</th
                        >
                    {/each}
                </tr>
            </thead>
            <tbody class="text-center">
                {#each rows as row, rowIndex}
                    <tr class="hover">
                        {#each row as item, colIndex}
                            {#if colIndex == 0 || !editable}
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
