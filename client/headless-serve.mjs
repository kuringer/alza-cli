// One headless Chromium (Playwright, no window) for many page scripts.
// stdin: one JSON request per line {"id", "url", "ready", "script"}.
// stdout: one JSON line per request {"id", "data"} or {"id", "error": {...}}.
// Requests run concurrently, each in its own page; the browser closes on stdin EOF.
import { chromium } from 'playwright';
import readline from 'node:readline';

const out = (o) => process.stdout.write(JSON.stringify(o) + '\n');
const fail = (id, e) => out({ id, error: { code: 'headless_error', message: String(e && e.message || e).slice(0, 500) } });

let ctx;
let browser;
try {
  browser = await chromium.launch({ headless: true, args: ['--disable-blink-features=AutomationControlled'] });
  ctx = await browser.newContext({
    locale: 'sk-SK',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36',
  });
} catch (e) {
  fail(0, e);
  process.exit(1);
}

async function run(req) {
  let page;
  try {
    page = await ctx.newPage();
    await page.goto(req.url, { waitUntil: 'domcontentloaded', timeout: 20000 });
    if (req.ready) {
      await page.waitForSelector(req.ready, { state: 'attached', timeout: 15000 }).catch(() => {});
    }
    // String expression: evaluated over CDP, not subject to page CSP.
    const data = await page.evaluate('(() => {' + req.script + '\n})()');
    out({ id: req.id, data });
  } catch (e) {
    fail(req.id, e);
  } finally {
    if (page) await page.close().catch(() => {});
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
  const p = run(req).finally(() => pending.delete(p));
  pending.add(p);
});
rl.on('close', async () => {
  await Promise.allSettled([...pending]);
  await browser.close();
  process.exit(0);
});
