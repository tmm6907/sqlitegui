<script lang="ts">
    let props = $props();
</script>

<li class="">
    <button
        id={`list-item-${props.identifier}`}
        class="block w-full
        border-b border-b-primary
        hover:bg-accent-content
        focus:bg-accent-content text-left"
        tabindex="0"
        onclick={() => {
            props.action();
        }}
    >
        <span
            class={`p-2 ${props.clamp !== undefined ? `line-clamp-${props.clamp}` : ""}`}
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
