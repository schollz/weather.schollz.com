import { access, readFile } from "node:fs/promises";

const outputDirectory = new URL("../dist/client/", import.meta.url);
const indexFile = new URL("index.html", outputDirectory);
const notFoundFile = new URL("404.html", outputDirectory);
const basePath = process.env.PAGES_BASE_PATH || "";

await access(indexFile);
await access(notFoundFile);

const html = await readFile(indexFile, "utf8");
const notFoundHtml = await readFile(notFoundFile, "utf8");
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

if (!notFoundHtml.includes("Finding your local weather")) {
  throw new Error("GitHub Pages fallback does not render the weather app.");
}

console.log("GitHub Pages static artifact verified.");
