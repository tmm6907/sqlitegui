<script lang="ts">
    import { fly } from "svelte/transition";
    import { alertStore } from "../stores/alertStore.ts";

    let msg = $state("");
    let type = $state("alert-success");
    let show = $state(false);
    alertStore.subscribe((alert) => {
        msg = alert.msg;
        type = alert.type;
        show = alert.show;
    });
</script>

{#if show}
    <div
        role="alert"
        class="alert {type} max-w-[48ch] fixed top-8 left-[85%] transform -translate-x-1/2 z-20 shadow-lg"
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
        <span>{msg}</span>
    </div>
{/if}
