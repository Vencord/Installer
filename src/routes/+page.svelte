<script lang="ts">
    import { invoke } from "@tauri-apps/api/tauri";
    import type { DiscordInstall } from "../types";

    const installs = invoke<DiscordInstall[]>("find_discords");

    let res = "-";
    async function install() {
        res = await invoke("install");
    }
    async function uninstall() {
        res = await invoke("uninstall");
    }
    async function repair() {
        res = await invoke("repair");
    }

    let selectedInstall: DiscordInstall;
    $: installs.then(i => (selectedInstall = i[0]));

    let customInstall: DiscordInstall;
    async function chooseCustomInstall() {
        customInstall = await invoke("pick_custom_install");
        console.log(customInstall);
    }
</script>

<div class="container">
    <h1>Vencord Installer</h1>

    <h3>Installs</h3>
    {#await installs}
        <p>Loading...</p>
    {:then installs}
        <div class="radios">
            {#each installs as install, idx}
                <label>
                    <input type="radio" name="install" value={install} bind:group={selectedInstall} />
                    <img src={`/${install.branch.toLowerCase()}.webp`} alt="" width="32" height="32" />
                    {install.branch} ({install.path})
                    {" "}
                    {install.is_patched ? "🟢" : "🔴"}
                </label>
            {/each}
            <label>
                <input
                    type="radio"
                    name="install"
                    value={null}
                    bind:group={selectedInstall}
                    on:change={chooseCustomInstall}
                />
                Pick Custom
            </label>
        </div>
    {:catch error}
        <p>{error.message}</p>
    {/await}

    <div class="buttons">
        <button class="btn-green" on:click={install}>Install</button>
        <button class="btn-red" on:click={uninstall}>Uninstall</button>
        <button class="btn-neutral" on:click={repair}>Repair</button>
    </div>
</div>

<style>
    :root {
        font-family: Inter, Avenir, Helvetica, Arial, sans-serif;
        font-size: 16px;
        line-height: 24px;
        font-weight: 400;

        color: #0f0f0f;
        background-color: #f6f6f6;

        font-synthesis: none;
        text-rendering: optimizeLegibility;
        -webkit-font-smoothing: antialiased;
        -moz-osx-font-smoothing: grayscale;
        -webkit-text-size-adjust: 100%;
    }

    .container {
        padding: 1em;
    }

    h1 {
        text-align: center;
    }

    .radios {
        display: grid;
        gap: 0.5em;
        margin: 1em 0;
    }

    label {
        display: flex;
        align-items: center;
        gap: 0.5em;
    }

    input[type="radio"] {
        height: 1.2rem;
        width: 1.2rem;
    }

    .buttons {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 1em;
    }

    button {
        padding: 0.75em;
        border-radius: 6px;
        border: none;
    }

    .btn-green {
        background-color: lime;
        color: black;
    }

    .btn-red {
        background-color: hotpink;
        color: black;
    }

    .btn-neutral {
        background-color: lightskyblue;
        color: black;
    }

    @media (prefers-color-scheme: dark) {
        :root {
            color: #f6f6f6;
            background-color: #2f2f2f;
        }
    }
</style>
