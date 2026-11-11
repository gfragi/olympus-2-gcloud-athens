# OLYMPUS 2.0 ⚡ — Google Cloud Day Athens 2026

![Google Cloud Day Athens 2026](logo.png)

A 2D cyber-Olympian arena shooter written in Go with [Ebitengine](https://ebitengine.org), built during the **"Vibe Code a 2D Game with Gemini and Antigravity 2.0"** workshop & Mini Game Jam at **Google Cloud Day Athens 2026** (November 10, 2026). It compiles to WebAssembly and is deployed on **Google Cloud Run**.

> Deprecated gods run the server-citadel above Athens. You are a rogue owl drone climbing to steal root access. Survive the purge, then take down **ZEUS.EXE**.

## Gameplay

| Control | Action |
|---|---|
| `W` `A` `S` `D` / arrow keys | Move |
| Mouse | Aim |
| Left click | Shoot cyan bolts |
| Click / `Space` | Start / restart |

- **Survive 15 seconds** of enemy waves, then **Zeus** descends. Dodge his telegraphed lightning volleys and purge him to win.
- **Terrain:** marble columns, rubble and server monoliths are randomly placed each run. They block movement *and* bolts, so use them as cover.
- **Enemies:**
  - **Purge Node**: spinning spiked chaser (1 HP)
  - **Harpy**: fast, weaving winged bot (1 HP)
  - **Minotaur**: horned brute that charges when you get close (3 HP)
- **Zeus.EXE:** crowned boss whose eyes track you. His core flashes magenta before he strikes, and he enrages at half HP and summons minions.
- **Power-ups** (spawn over time and drop from enemies):
  - **Ambrosia**: +1 HP
  - **Aegis**: 6s immunity; enemies that touch you shatter
  - **Owl Sight**: 7s triple shot

Everything (graphics and sound) is procedural, so there are no asset files.

## Run locally

Requires Go 1.26+.

```bash
# Native desktop window
go run .

# Or in the browser, exactly as on Cloud Run
docker build -t olympus .
docker run --rm -p 8080:8080 olympus
# open http://localhost:8080
```

## Deploy to Google Cloud Run

The `Dockerfile` builds the game to WebAssembly and serves it with a tiny Go static server (with a `/healthz` endpoint).

```bash
gcloud run deploy olympus-2 \
  --source . \
  --region europe-west1 \
  --allow-unauthenticated
```

## Repository structure

```
main.go          # the whole game (Ebitengine)
public/          # web shell: index.html + wasm_exec.js (game.wasm is built by Docker)
Dockerfile       # WASM build + minimal Cloud Run server
check_env.sh     # workshop environment checker
```

## Credits

- Built by [George Fragiadakis](https://github.com/gfragi) at Google Cloud Day Athens 2026.
- Workshop materials by [Daniela Petruzalek](https://github.com/danicat): [danicat/cd-athens-2026](https://github.com/danicat/cd-athens-2026).

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).

## Disclaimer

This is not an officially supported Google product. It is an experimental project created for educational purposes.
