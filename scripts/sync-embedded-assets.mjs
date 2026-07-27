import { cp, mkdir, readdir, rm } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const source = resolve(root, "dist/client");
const destination = resolve(root, "internal/web/public");

await mkdir(destination, { recursive: true });

for (const entry of await readdir(destination)) {
  if (entry !== "placeholder.txt") {
    await rm(resolve(destination, entry), { force: true, recursive: true });
  }
}

await cp(source, destination, { recursive: true });
