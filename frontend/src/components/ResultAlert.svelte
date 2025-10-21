<script lang="ts">
    import { resultAlertStore } from "../stores/alertStore.ts";

    let msg = $state("");
    let type = $state("success");
    let show = $state(false);
    // Subscribe to the store
    resultAlertStore.subscribe((alert) => {
        msg = alert.msg;
        type = alert.type;
        if (alert.show) {
            show = true;
        } else {
            show = false;
        }
    });
</script>

<div
    class="
        absolute top-[-0.5em] right-0
        flex gap-2 justify-between items-center
        min-w-[16ch] z-20
        px-4 py-1 rounded
        bg-{type} font-bold text-neutral
        transition-opacity duration-300
        
        {show ? 'opacity-100 visible' : 'opacity-0 invisible'}
    "
>
    <div>
        <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            class="stroke-current h-6 w-6 shrink-0"
            height="24"
            width="24"
        >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            ></path>
        </svg>
    </div>
    <div>
        <span class="text-sm">{msg}</span>
    </div>
</div>
