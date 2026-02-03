import { cp, mkdir, stat } from "node:fs/promises";
import { resolve } from "node:path";

const sourceDir = resolve("public/assets");
const targetDir = resolve("../internal/webassets/dist/assets");

try {
  await stat(sourceDir);
} catch (error) {
  console.error(`Source assets directory not found: ${sourceDir}`);
  process.exit(1);
}

await mkdir(targetDir, { recursive: true });
await cp(sourceDir, targetDir, { recursive: true });
console.log(`Synced assets from ${sourceDir} to ${targetDir}`);
