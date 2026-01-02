import fs from "fs";
import os from "os";
import path from "path";
import { chromium } from "playwright";

const configDir = path.join(os.homedir(), ".config", "alza");
const profileDir =
  process.env.ALZA_PROFILE_DIR || path.join(configDir, "pw-profile");
const headful =
  process.env.HEADFUL === "1" || process.env.ALZA_HEADFUL === "1";
const timeoutMs = Number(process.env.ALZA_TOKEN_TIMEOUT_MS || "180000");
const startUrl = process.env.ALZA_URL || "https://www.alza.sk/";

const launchOptions = {
  headless: !headful,
  args: [],
};

if (!headful) {
  launchOptions.args.push("--no-sandbox");
}

if (process.env.ALZA_CHROMIUM_PATH) {
  launchOptions.executablePath = process.env.ALZA_CHROMIUM_PATH;
}

const context = await chromium.launchPersistentContext(
  profileDir,
  launchOptions,
);
const page = context.pages()[0] || (await context.newPage());

let authToken = "";
const isValidAuth = (value) =>
  value &&
  value.startsWith("Bearer ") &&
  !value.includes("undefined") &&
  value.length > 40;
page.on("request", (request) => {
  const auth = request.headers()["authorization"];
  if (!authToken && isValidAuth(auth)) {
    authToken = auth;
  }
});

console.log("Opening Alza...");
await page.goto(startUrl, { waitUntil: "domcontentloaded" });
console.log("If login is required, complete it in the browser.");
console.log("After login, type into the search box to trigger token capture.");

const deadline = Date.now() + timeoutMs;
while (!authToken && Date.now() < deadline) {
  await page.waitForTimeout(500);
}

if (!authToken) {
  console.error("✗ Could not capture token - you may need to log in again");
  await context.close();
  process.exit(1);
}

fs.mkdirSync(configDir, { recursive: true });
fs.writeFileSync(path.join(configDir, "auth_token.txt"), authToken, {
  mode: 0o600,
});
console.log("✓ Token refreshed and saved to ~/.config/alza/auth_token.txt");

await context.close();
