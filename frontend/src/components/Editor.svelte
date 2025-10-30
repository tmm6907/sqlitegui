<script lang="ts">
    import { onDestroy, onMount } from "svelte";
    import { EditorState } from "@codemirror/state";
    import { drawSelection, EditorView, keymap } from "@codemirror/view";
    import {
        defaultKeymap,
        history,
        historyKeymap,
    } from "@codemirror/commands";
    import { sql } from "@codemirror/lang-sql";
    import { tags } from "@lezer/highlight";
    import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
    import {
        renderNavDataWithResultAlert,
        triggerResultAlert,
    } from "../stores/alertStore.ts";
    import {
        dbNameStore,
        loadingResultsStore,
        queryResults,
    } from "../stores/resultsStore.ts";
    import { Query } from "../../wailsjs/go/main/App.js";
    import ListItem from "./ListItem.svelte";
    import { nonpassive } from "svelte/legacy";

    let editorView: EditorView;
    let dbName = $state("");

    let queries = $state([]);
    let loadingResults = $state(true);

    loadingResultsStore.subscribe((val) => {
        loadingResults = val;
        console.log(loadingResults);
    });

    dbNameStore.subscribe((name) => (dbName = name));

    async function handleKeyDown(ev: KeyboardEvent) {
        if (
            ev.key &&
            ev.key.startsWith("F") &&
            ev.key.length >= 2 &&
            ev.key.length <= 3
        ) {
            ev.preventDefault();

            console.log(`Function Key Captured: ${ev.key}`);

            // Example: Call a Go function when F1 is pressed
            if (ev.key === "F5") {
                await runQuery();
            }
        }
    }

    async function runQuery() {
        if (!editorView) {
            console.error("EditorView is not initialized yet.");
            return;
        }
        let query = editorView.state.doc.toString();
        if (query.length === 0) {
            triggerResultAlert("Query cannot be empty!", "error");
            return;
        }
        loadingResultsStore.set(true);
        let res = await Query({ query: query, editable: false });
        loadingResultsStore.set(false);
        if (res.error !== "" || undefined) {
            triggerResultAlert(res.error, "error");
            console.error(res.error);
            return;
        }
        let results = res.results;
        if (results) {
            queryResults.set({
                pk: results.pk,
                cols: results.cols,
                rows: results.rows,
                editable: false,
            });
            renderNavDataWithResultAlert("Query successful!");
        } else {
            let msg = results.rowsAffected
                ? "Rows affected: " + results.rowsAffected
                : "";
            queryResults.set({
                pk: false,
                cols: [],
                rows: [],
                editable: false,
            });
            renderNavDataWithResultAlert(msg);
        }
        if (!queries.includes(query)) {
            queries = [query, ...queries];
        }

        editorView.dispatch({
            changes: {
                from: 0,
                to: editorView.state.doc.length,
                insert: "",
            },
        });
    }

    function enterRecentQuery(i: number) {
        let el = document.getElementById(`list-item-${i}`);
        if (!el) {
            console.error("failed to select recent query");
            triggerResultAlert("Failed to select recent query!", "alert-error");
        }
        editorView.dispatch({
            changes: {
                from: 0,
                to: editorView.state.doc.length,
                insert: el.textContent,
            },
        });
    }

    onMount(() => {
        loadingResults = false;
        let customHighlightStyle = HighlightStyle.define([
            { tag: tags.keyword, color: "var(--editor-primary)" },
            { tag: tags.string, color: "var(--editor-accent)" },
        ]);

        let parentElement = document.getElementById("sql-editor");
        if (!parentElement) {
            console.error("Parent element with id 'sql-editor' not found.");
            return;
        }
        const myCustomTheme = EditorView.theme({
            ".cm-activeLine": {
                backgroundColor: "#3c3c3c",
            },
            ".cm-lineNumbers .cm-gutterElement": {
                color: "#6d6d6d",
            },
            ".cm-scroller": {
                overflow: "none",
            },
            ".cm-line": {
                "text-wrap": "wrap",
            },
        });

        let startState = EditorState.create({
            doc: "",
            extensions: [
                sql(),
                history(),
                keymap.of([...defaultKeymap, ...historyKeymap]),
                syntaxHighlighting(customHighlightStyle),
                myCustomTheme,
                EditorView.lineWrapping,
            ],
        });

        editorView = new EditorView({
            state: startState,
            parent: parentElement,
            extensions: [drawSelection()],
        });
        editorView.focus();
        document.addEventListener("keydown", handleKeyDown);
    });
    onDestroy(() => document.removeEventListener("keydown", handleKeyDown));
</script>

<div class="flex space-x-8 h-full">
    <div class="w-2/3 flex flex-col space-y-2">
        <div class="flex justify-between items-center">
            <label for="sql-editor">Editor</label>
            <button class="btn btn-xs btn-neutral" onclick={runQuery} title="F5"
                >Run Query<i class="fa-solid fa-play text-success"></i></button
            >
        </div>
        <div
            id="sql-editor"
            class="h-full w-full textarea textarea-accent **:dark:caret-base-content"
        ></div>
    </div>
    <div class="{loadingResults ? '' : 'invisible'} flex place-items-center">
        <span class="loading loading-spinner loading-md"></span>
    </div>
    <div class="flex-1 flex flex-col space-y-2 grid-rows-[auto_1fr]">
        <label for="recent-queries">Recent Queries</label>
        <div class="outline rounded h-full overflow-auto">
            <ul id="recent-queries" class="list">
                {#each queries as q, i}
                    <ListItem
                        identifier={i}
                        text={q}
                        clamp={2}
                        action={() => {
                            enterRecentQuery(i);
                        }}
                    />
                {/each}
            </ul>
        </div>
    </div>
</div>
