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
        absolute top-[-1.5em] right-0
        min-w-[16ch] z-20
        alert
        alert-{type}
        transition-opacity duration-300
        {show ? 'opacity-100 visible' : 'opacity-0 invisible'}
    "
>
    <div>
        <i class="fa-solid fa-info"></i>
    </div>
    <div>
        <span class="text-sm">{msg}</span>
    </div>
</div>
