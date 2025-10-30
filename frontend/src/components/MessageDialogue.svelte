<script lang="ts">
    import { msgDialogueStore } from "src/stores/dialogueStore.ts";
    type DialogueOption = string[];
    let dialogueTitle = $state("");
    let msg = $state("Test");
    let options: DialogueOption = $state(["this", "that"]);
    let actions = $state([]);
    let show = $state(false);
    let btnStyles = $state([]);
    msgDialogueStore.subscribe((val) => {
        dialogueTitle = val.title;
        msg = val.msg;
        options = val.options;
        if (
            val.actions.length > 0 &&
            val.actions.length == val.options.length
        ) {
            actions = val.actions;
        }
        show = val.show;
        btnStyles = val.btnStyles;
    });
</script>

{#if show}
    <dialog
        open
        class="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 flex flex-col w-1/3 min-h-4/12 bg-base-200 p-8 rounded-lg border border-primary z-20"
    >
        <h3 class="text-2xl font-bold mb-8">{dialogueTitle}</h3>
        <div class="max-h-32 overflow-auto">
            <p>
                {msg}
            </p>
        </div>
        <div class="flex space-x-8 mt-auto justify-center">
            {#each options as opt, index}
                {#if actions[index]}
                    <button
                        class={`btn btn-lg ${btnStyles && btnStyles[index] ? btnStyles[index] : "btn-ghost"}`}
                        onclick={async () => {
                            await actions[index]();
                            msgDialogueStore.set({
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
                        class={`btn btn-lg ${btnStyles && btnStyles[index] ? btnStyles[index] : "btn-ghost"}`}
                        >{opt}</button
                    >
                {/if}
            {/each}
        </div>
    </dialog>
{/if}

<style>
</style>
