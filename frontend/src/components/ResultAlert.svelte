<script lang="ts">
    import { appState } from "src/stores/appState.svelte.ts";
    $effect(() => {
        if (appState.resultAlert.show) {
            const timerId = setTimeout(() => {
                appState.resultAlert.duration = 3000;
                appState.resultAlert.msg = "";
                appState.resultAlert.show = false;
            }, appState.resultAlert.duration);
            return () => {
                clearTimeout(timerId);
            };
        }
    });
</script>

<div
    class={`
        absolute top-[-1.5em] right-0
        min-w-[16ch] z-20
        alert
        alert-${appState.resultAlert.type}
        transition-opacity duration-300
        ${
            appState.resultAlert.show
                ? "opacity-100 visible"
                : "opacity-0 invisible"
        }
    `}
>
    <div>
        <i class="fa-solid fa-info"></i>
    </div>
    <div>
        <span class="text-sm">{appState.resultAlert.msg}</span>
    </div>
</div>
