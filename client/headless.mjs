// Runs a page script in headless Chromium (Playwright, no window) and prints
// {"data": <result>} or {"error": {...}}. Args: <url> <readySelector>; script on stdin.
import { chromium } from 'playwright';

const [url, ready] = process.argv.slice(2);
const chunks = [];
for await (const c of process.stdin) chunks.push(c);
const script = Buffer.concat(chunks).toString('utf8');

const out = (o) => process.stdout.write(JSON.stringify(o) + '\n');
let browser;
try {
  browser = await chromium.launch({ headless: true, args: ['--disable-blink-features=AutomationControlled'] });
  const ctx = await browser.newContext({
    locale: 'sk-SK',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36',
  });
  const page = await ctx.newPage();
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 20000 });
  if (ready) {
    await page.waitForSelector(ready, { state: 'attached', timeout: 15000 }).catch(() => {});
  }
  // String expression: evaluated over CDP, not subject to page CSP.
  const data = await page.evaluate('(() => {' + script + '\n})()');
  out({ data });
} catch (e) {
  out({ error: { code: 'headless_error', message: String(e && e.message || e).slice(0, 500) } });
} finally {
  if (browser) await browser.close();
}
