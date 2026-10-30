// One headless Chromium (Playwright, no window) for many page scripts.
// stdin: one JSON request per line {"id", "url", "ready", "script", "timeoutMs"},
// or {"id", "cancel": true} to abort a running request.
// stdout: one JSON line per request {"id", "data"} or {"id", "error": {...}}.
// Requests run concurrently; the browser closes on stdin EOF. Each request gets
// a fresh context (no shared cookies): Heureka's bot check (DataDome) answers
// the second page of one cookie session with a challenge.
import { chromium } from 'playwright';
import readline from 'node:readline';

const out = (o) => process.stdout.write(JSON.stringify(o) + '\n');
const fail = (id, e) => out({ id, error: { code: 'headless_error', message: String(e && e.message || e).slice(0, 500) } });

const contextOptions = {
  locale: 'sk-SK',
  userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36',
};
let browser;
try {
  browser = await chromium.launch({ headless: true, args: ['--disable-blink-features=AutomationControlled'] });
} catch (e) {
  fail(0, e);
  process.exit(1);
}

// Total budget of one request (navigation + readiness + script) when the
// client sends no timeoutMs. The Go side sends one below its own deadline.
const defaultTimeoutMs = 25000;
// id -> browser context of running requests; closing it aborts the page.
const active = new Map();
const cancelled = new Set();
const running = new Set();

async function run(req) {
  let ctx;
  let timer;
  running.add(req.id);
  const ms = req.timeoutMs > 0 ? req.timeoutMs : defaultTimeoutMs;
  const t0 = Date.now();
  const left = () => Math.max(1, ms - (Date.now() - t0));
  const deadline = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`deadline ${ms} ms exceeded`)), ms);
  });
  const work = (async () => {
    ctx = await browser.newContext(contextOptions);
    active.set(req.id, ctx);
    if (cancelled.has(req.id)) throw new Error('cancelled');
    const page = await ctx.newPage();
    await page.goto(req.url, { waitUntil: 'domcontentloaded', timeout: left() });
    if (req.ready) {
      await page.waitForSelector(req.ready, { state: 'attached', timeout: Math.min(15000, left()) }).catch(() => {});
    }
    // String expression: evaluated over CDP, not subject to page CSP.
    return page.evaluate('(() => {' + req.script + '\n})()');
  })();
  work.catch(() => {}); // the deadline may win the race; avoid an unhandled rejection
  try {
    const data = await Promise.race([work, deadline]);
    out({ id: req.id, data });
  } catch (e) {
    fail(req.id, cancelled.has(req.id) ? 'cancelled' : e);
  } finally {
    clearTimeout(timer);
    running.delete(req.id);
    active.delete(req.id);
    cancelled.delete(req.id);
    // Closing the context also stops a page or evaluate still running.
    if (ctx) await ctx.close().catch(() => {});
  }
}

const pending = new Set();
const rl = readline.createInterface({ input: process.stdin });
rl.on('line', (line) => {
  if (!line.trim()) return;
  let req;
  try {
    req = JSON.parse(line);
  } catch (e) {
    fail(0, e);
    return;
  }
  if (req.cancel) {
    // The client gave up on this request: free its page now.
    if (!running.has(req.id)) return;
    cancelled.add(req.id);
    const c = active.get(req.id);
    if (c) c.close().catch(() => {});
    return;
  }
  const p = run(req).finally(() => pending.delete(p));
  pending.add(p);
});
rl.on('close', async () => {
  await Promise.allSettled([...pending]);
  await browser.close();
  process.exit(0);
});
