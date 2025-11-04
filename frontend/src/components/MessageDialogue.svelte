<script lang="ts">
    import { appState } from "src/stores/appState.svelte.ts";
    import { setDialogMsg } from "src/utils/utils.ts";
</script>

{#if appState.dialog.show}
    <dialog
        open
        class="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 flex flex-col w-1/3 min-h-4/12 bg-base-200 p-8 rounded-lg border border-primary z-20"
    >
        <h3 class="text-2xl font-bold mb-8">{appState.dialog.title}</h3>
        <div class="max-h-32 overflow-auto">
            <p>
                {appState.dialog.msg}
            </p>
        </div>
        <div class="flex space-x-8 mt-auto justify-center">
            {#each appState.dialog.options as opt, index}
                {#if appState.dialog.actions[index]}
                    <button
                        class={`btn btn-lg ${appState.dialog.btnStyles && appState.dialog.btnStyles[index] ? appState.dialog.btnStyles[index] : "btn-ghost"}`}
                        onclick={async () => {
                            await appState.dialog.actions[index]();
                            setDialogMsg({
                                title: "",
                                msg: "",
                                options: [],
                                actions: [],
                                show: false,
                                btnStyles: [],
                            });
                        }}>{opt}</button
                    >
                {:else}
                    <button
                        class={`btn btn-lg ${appState.dialog.btnStyles && appState.dialog.btnStyles[index] ? appState.dialog.btnStyles[index] : "btn-ghost"}`}
                        >{opt}</button
                    >
                {/if}
            {/each}
        </div>
    </dialog>
{/if}

<style>
</style>
