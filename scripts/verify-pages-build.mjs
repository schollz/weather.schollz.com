import { access, readFile } from "node:fs/promises";

const outputDirectory = new URL("../dist/client/", import.meta.url);
const indexFile = new URL("index.html", outputDirectory);
const basePath = process.env.PAGES_BASE_PATH || "";

await access(indexFile);

const html = await readFile(indexFile, "utf8");
const assetReferences = [
  ...html.matchAll(/(?:href|src)="([^"]*\/assets\/[^"]+)"/g),
].map((match) => match[1]);

if (!assetReferences.length) {
  throw new Error("GitHub Pages build has no static asset references.");
}

if (
  basePath &&
  assetReferences.some(
    (reference) => !reference.startsWith(`${basePath}/assets/`),
  )
) {
  throw new Error(
    `GitHub Pages assets were not prefixed with ${basePath}.`,
  );
}

console.log("GitHub Pages static artifact verified.");
