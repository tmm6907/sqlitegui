<script lang="ts">
    let props = $props();
</script>

<li class=" py-2 border-b-primary-content">
    <button
        id={`list-item-${props.identifier}`}
        class="btn btn-block btn-ghost text-left"
        tabindex="0"
        onclick={() => {
            props.action();
        }}
    >
        <span
            class={`w-full ${props.clamp !== undefined ? `line-clamp-${props.clamp}` : ""}`}
            title={props.text}>{props.text}</span
        >
    </button>
</li>

<style>
    .line-clamp-3 {
        overflow: hidden;
        display: -webkit-box;
        -webkit-box-orient: vertical;
        -webkit-line-clamp: 3;
        line-clamp: 3;
        /* Optional: Ensure consistent height/line spacing */
        line-height: 1.4;
        max-height: calc(1.4em * 3);
    }
    /* You can define line-clamp-2, line-clamp-4, etc., here if needed */
    .line-clamp-2 {
        overflow: hidden;
        display: -webkit-box;
        -webkit-box-orient: vertical;
        -webkit-line-clamp: 2;
        line-clamp: 2;
        line-height: 1.4;
        max-height: calc(1.4em * 2);
    }
</style>
