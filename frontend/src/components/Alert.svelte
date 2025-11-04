<script lang="ts">
    import { fly } from "svelte/transition";
    import { appState } from "src/stores/appState.svelte.ts";

    $effect(() => {
        if (appState.alert.show) {
            console.log(appState.alert.type);
            const timerId = setTimeout(() => {
                appState.alert.duration = 3000;
                appState.alert.msg = "";
                appState.alert.show = false;
            }, appState.alert.duration);
            return () => {
                clearTimeout(timerId);
            };
        }
    });
</script>

{#if appState.alert.show}
    <div
        role="alert"
        class={`
        alert alert-${appState.alert.type} 
        max-w-[48ch] fixed top-4 right-8 
        transform z-20 shadow-lg
    `}
        transition:fly={{ y: -50, duration: 500 }}
    >
        <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            class="stroke-current h-6 w-6 shrink-0"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            ></path>
        </svg>
        <span>{appState.alert.msg}</span>
    </div>
{/if}
