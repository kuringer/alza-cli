import fs from "fs";
import os from "os";
import path from "path";

const configDir = path.join(os.homedir(), ".config", "alza");
const cdpBaseUrl = process.env.ALZA_CDP_URL || "http://127.0.0.1:9222";
const timeoutMs = Number(process.env.ALZA_TOKEN_TIMEOUT_MS || "900000");
const pollMs = Number(process.env.ALZA_TOKEN_POLL_MS || "2000");
const tokenUrl =
  process.env.ALZA_TOKEN_URL ||
  "https://www.alza.sk/api/identity/v1/accesstoken";

const start = Date.now();

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

const saveToken = (token) => {
  fs.mkdirSync(configDir, { recursive: true });
  fs.writeFileSync(path.join(configDir, "auth_token.txt"), token, {
    mode: 0o600,
  });
};

const fetchTargets = async () => {
  const resp = await fetch(`${cdpBaseUrl}/json/list`);
  if (!resp.ok) {
    throw new Error(`CDP list status ${resp.status}`);
  }
  return await resp.json();
};

const pickTarget = (targets) =>
  targets.find(
    (t) => t.type === "page" && t.url && t.url.includes("alza.sk"),
  );

const connectCDP = (wsUrl) =>
  new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl);
    const pending = new Map();
    let id = 0;

    const cleanup = () => {
      pending.clear();
      ws.close();
    };

    const send = (method, params = {}) =>
      new Promise((res, rej) => {
        const requestId = ++id;
        const timeout = setTimeout(() => {
          pending.delete(requestId);
          rej(new Error(`CDP timeout for ${method}`));
        }, 10000);
        pending.set(requestId, (msg) => {
          clearTimeout(timeout);
          pending.delete(requestId);
          res(msg);
        });
        ws.send(JSON.stringify({ id: requestId, method, params }));
      });

    ws.addEventListener("open", () => {
      resolve({ send, close: cleanup });
    });

    ws.addEventListener("message", (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.id && pending.has(msg.id)) {
          pending.get(msg.id)(msg);
        }
      } catch {
        // ignore malformed messages
      }
    });

    ws.addEventListener("error", () => {
      cleanup();
      reject(new Error("CDP websocket error"));
    });

    ws.addEventListener("close", () => {
      cleanup();
    });
  });

let debugLogged = false;

while (Date.now() - start < timeoutMs) {
  try {
    const targets = await fetchTargets();
    const target = pickTarget(targets);
    if (!target?.webSocketDebuggerUrl) {
      console.log("No Alza page yet - waiting for login...");
      await sleep(pollMs);
      continue;
    }

    const cdp = await connectCDP(target.webSocketDebuggerUrl);
    const expr = `(async () => {
      const r = await fetch(${JSON.stringify(tokenUrl)}, { credentials: "include" });
      const body = await r.text();
      const contentType = r.headers.get("content-type") || "";
      return JSON.stringify({ status: r.status, contentType, body });
    })()`;

    const result = await cdp.send("Runtime.evaluate", {
      expression: expr,
      awaitPromise: true,
      returnByValue: true,
    });
    cdp.close();

    if (result.exceptionDetails) {
      console.log("Eval failed - waiting...");
      await sleep(pollMs);
      continue;
    }

    if (result.error) {
      console.log(`CDP error: ${result.error.message || "unknown"} - waiting...`);
      await sleep(pollMs);
      continue;
    }

    const payload = result.result?.result?.value;
    if (typeof payload !== "string" || !payload) {
      const kind = result.result?.type || "unknown";
      if (process.env.ALZA_DEBUG === "1" && !debugLogged) {
        console.log(`CDP raw response: ${JSON.stringify(result).slice(0, 500)}`);
        debugLogged = true;
      }
      console.log(`Empty token response (type=${kind}) - waiting...`);
      await sleep(pollMs);
      continue;
    }

    let data;
    let rawBody = "";
    try {
      const parsed = JSON.parse(payload);
      if (!parsed?.body) {
        console.log(
          `Token endpoint status ${parsed?.status ?? "?"} (${parsed?.contentType ?? "unknown"}) - waiting...`,
        );
        await sleep(pollMs);
        continue;
      }
      rawBody = parsed.body;
      data = JSON.parse(rawBody);
    } catch {
      const snippet = rawBody ? rawBody.slice(0, 200) : "(empty)";
      console.log(`Non-JSON token response (${snippet}) - waiting...`);
      await sleep(pollMs);
      continue;
    }

    const accessToken = data?.AccessToken || data?.accessToken;
    const logOut = data?.LogOut ?? data?.logOut;
    if (accessToken && !logOut) {
      const token = `Bearer ${accessToken}`;
      saveToken(token);
      console.log("✓ Token refreshed and saved to ~/.config/alza/auth_token.txt");
      process.exit(0);
    }

    console.log("No access token yet - waiting...");
  } catch (err) {
    console.log("Waiting for browser...", err?.message || err);
  }

  await sleep(pollMs);
}

console.error("✗ Timeout waiting for login/token");
process.exit(1);
