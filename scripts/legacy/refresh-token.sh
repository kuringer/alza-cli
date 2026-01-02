#!/bin/bash
# Deprecated: use `alza token refresh` (Chrome cookies) or scripts/legacy/refresh-token-cdp.sh

echo "Deprecated: use 'alza token refresh' or ./scripts/legacy/refresh-token-cdp.sh instead." >&2
exit 1

cd ~/.claude/skills/dev-browser

bun x tsx << 'EOF'
import { connect, waitForPageLoad } from "./src/client.ts";
import fs from 'fs';
import os from 'os';
import path from 'path';

const client = await connect();
const page = await client.page("alza-refresh");

console.log("Refreshing Alza auth token...");

await page.goto('https://www.alza.sk/');
await waitForPageLoad(page);

let authToken = null;

page.on('request', request => {
  const auth = request.headers()['authorization'];
  if (auth && auth.startsWith('Bearer ') && !authToken) {
    authToken = auth;
  }
});

// Trigger API call by typing in search
await page.fill('input[name="exps"]', 'test');
await page.waitForTimeout(2000);

if (authToken) {
  const configDir = path.join(os.homedir(), '.config', 'alza');
  fs.mkdirSync(configDir, { recursive: true });
  fs.writeFileSync(path.join(configDir, 'auth_token.txt'), authToken);
  console.log("✓ Token refreshed and saved");
} else {
  console.log("✗ Could not capture token - you may need to log in again");
  process.exit(1);
}

await client.disconnect();
EOF
