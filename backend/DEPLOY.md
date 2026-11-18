# Deploying the MindSuite agent backend

This is for DevOps. It explains how to run the MindSuite agent backend on a server and how to give it the Anthropic (Claude) API key.

The web app is static and is already hosted on GitHub Pages: <https://mjaffry01.github.io/mindsuite-doctrine-of-legacy-modernization/>. The backend is a separate Go program that the web app calls.

## 1. The Claude API key

The backend reads the key from the environment variable **`ANTHROPIC_API_KEY`**. If that variable is set, it is used and nothing else is looked at.

1. Store the key in your platform's secret store, for example:
   - Azure Key Vault (referenced from App Service or Container Apps settings);
   - AWS Secrets Manager (attached to the ECS task or Lambda);
   - a Kubernetes Secret;
   - your CI/CD system's secret variables;
   - on a plain Linux VM with systemd, an `EnvironmentFile` owned by root and readable only by the service account (`chmod 600`).
2. Expose it to the backend process as `ANTHROPIC_API_KEY`, spelled exactly like that.
3. After the backend starts, check the log line. It names where the key came from, never the key itself:
   ```
   MindSuite agent backend 0.1.0 on http://0.0.0.0:8090  · Claude: available (claude-opus-5-5, claude-sonnet-5-5; key from ANTHROPIC_API_KEY)
   ```
   `Claude: off` means the variable did not reach the process.

**Rules for the key**

- Never put it in the image, the repository, `server.json` / `mindsuite.json`, build output or logs. The backend refuses to start if its config file contains something that looks like a key.
- Use a key made only for this server, with its own name and a monthly spending limit in the Anthropic Console (console.anthropic.com). The best way is for DevOps to create it in the organisation's Anthropic account themselves, so the key is never passed by hand.
- If it must be handed over, use a password manager's secure sharing. Never use e-mail, chat or tickets.
- If the key may have leaked, revoke it in the Anthropic Console and store a new one.

Without a key the backend still works: the rule-based agents run and the Claude-powered agents are skipped.

## 2. Build

The backend needs Go 1.26.3 or later and has no other runtime dependencies.

```bash
cd backend
go build -o mindsuite-agents ./cmd/mindsuite-agents
```

The result is a single program. Copy it to the server together with your `server.json`.

## 3. Settings: `server.json`

Copy [`server.example.json`](server.example.json) to `server.json` and adjust it.

| Field | Server value | Meaning |
|---|---|---|
| `addr` | `0.0.0.0:8090` | Address and port to listen on. Put the reverse proxy in front (section 4). |
| `runs` | a persistent folder, e.g. `/var/lib/mindsuite/runs` | Saved run results. They describe customer systems: keep the folder private and backed up, and decide how long to keep it. `""` keeps results in memory only. |
| `cors` | `["https://mjaffry01.github.io"]` | Browser origins allowed to call the API. Add your own web-app address if you host it elsewhere. |
| `allowPath` | `false` | Runs that read a folder on the server. Always off on a server (the backend also forces it off when not listening on localhost). |
| `models` | leave as is | The Claude model for each tier. Keep `strong` and `balanced` different: the verifier must not check a model with itself. |
| `apiKeyFile` | `""` | Only for a developer's own Windows laptop. Leave empty on a server. |

Start it with:

```bash
ANTHROPIC_API_KEY=… ./mindsuite-agents serve -config server.json
```

(In practice the platform sets `ANTHROPIC_API_KEY`; do not type the key on the command line, where it ends up in the shell history.) Command-line flags such as `-addr` or `-runs` override the file.

## 4. In front of the backend: HTTPS and login (required)

The backend has **no login of its own**. When it listens beyond localhost it prints a warning. Anyone who can reach it can send data for analysis and spend the Claude credits. Before it is reachable from a network:

1. **Login.** Put it behind your platform's authentication: Azure App Service / Container Apps authentication, AWS ALB with Cognito or OIDC, Cloudflare Access, or a reverse proxy with single sign-on. Only your own people (or the customer's) should get in.
2. **HTTPS.** The web app runs on `https://`, and browsers block it from calling an `http://` backend.
3. **Reverse-proxy settings:**
   - allow uploads up to **512 MB** (for example nginx `client_max_body_size 512m;`);
   - the live run log uses Server-Sent Events on `GET /api/runs/{id}/events`: turn response buffering off for it (nginx `proxy_buffering off;`) and allow long-lived connections (`proxy_read_timeout 1h;`);
   - pass the `X-MindSuite` request header through. Every `POST` must carry it.

## 5. Check it works

```bash
curl https://<your-backend>/api/health
```

The answer looks like this. `modelAvailable: true` means the key was found.

```json
{"ok":true,"version":"0.1.0","modelAvailable":true,"allowLocalPath":false,"models":{"strong":"claude-opus-5-5","balanced":"claude-sonnet-5-5","fast":"claude-haiku-5-5"},"teams":["batch","code","database","documents","identity","incidents","infrastructure","messaging","network","telemetry"]}
```

Then, in the web app, open a team panel, set **Agent backend address** to `https://<your-backend>` and press **Check connection**.

## 6. Privacy: decide before going live

The web app tells customers that their files are read in memory and that Claude is used only when they tick the consent box for that run. On a laptop or a server inside the customer's network, nothing else leaves their machine. On a shared server that you run, customers' exports leave their network and come to yours.

- **Preferred for sensitive customers:** run one backend per customer, inside their own network or cloud account.
- **Shared server:** agree on this with the product owner first. The privacy wording and consent text in the web app must be updated, and the `runs` folder must be protected and cleaned up on a schedule.

Data sent to Claude is always limited to what the consent box describes, and every request is listed in the run's `sentToModel`.

## 7. Updating

Rebuild the program from the new code, replace it on the server and restart. `server.json` and the `runs` folder stay as they are. If the key changes, update the secret in the secret store and restart the backend.
